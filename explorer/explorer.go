package explorer

import (
	"embed"
	"encoding/hex"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/fatih/color"
	"github.com/gorilla/mux"
	"github.com/liamzebedee/tinychain-go/core/nakamoto"
)

//go:embed templates/*.html assets/*
var embedFS embed.FS

type BlockExplorerServer struct {
	router      *mux.Router
	log         *log.Logger

	host string
	port        int
	environment string
	
	dag         *nakamoto.BlockDAG
	state 	 *nakamoto.StateMachine
}

type localDirFS struct {
	baseDir string
}
func (fs localDirFS) Open(name string) (fs.File, error) {
	return os.Open(filepath.Join(fs.baseDir, name))
}

func (expl *BlockExplorerServer) getFS() *fs.FS {
	_, currentFile, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(currentFile)
	
	fs := map[string]fs.FS{
		"dev": localDirFS{baseDir: baseDir},
		"test": embedFS,
		"live": embedFS,
	}[expl.environment]

	return &fs
}

func (expl *BlockExplorerServer) getTemplates(patterns ...string) *template.Template {
	fs := *expl.getFS()
	return template.Must(template.New("").ParseFS(fs, patterns...))
}

func NewBlockExplorerServer(dag *nakamoto.BlockDAG, port int) *BlockExplorerServer {
	log := NewLogger("explorer", "")
	environment := os.Getenv("ENV")
	if environment == "" {
		environment = "dev"
	}
	if !(environment == "dev" || environment == "test" || environment == "live") {
		log.Fatalf("Invalid environment %s, must be one of (dev, test, live)", environment)
	}

	log.Println("Environment:", environment)
	host := map[string]string{
		"dev": "127.0.0.1",
		"test": "0.0.0.0",
		"live": "0.0.0.0",
	}[environment]

	router := mux.NewRouter()

	expl := &BlockExplorerServer{
		router:      router,
		log:         log,
		host: host,
		port:        port,
		environment: environment,
		dag:         dag,
	}

	expl.router.HandleFunc("/", expl.homePage)
	expl.router.HandleFunc("/blocks/", expl.getChain)
	expl.router.HandleFunc("/blocks/{id}", expl.getBlock)
	expl.router.HandleFunc("/epochs/{id}", expl.getEpoch)
	expl.router.HandleFunc("/accounts/", expl.getAccounts)
	expl.router.HandleFunc("/accounts/{id}", expl.getAccount)
	expl.router.HandleFunc("/transactions/{id}", expl.getTransaction)

	// Serve static files.
	expl.router.PathPrefix("/assets/").HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		fs := *expl.getFS()
		httpFs := http.FS(fs)
		http.FileServer(httpFs).ServeHTTP(w, r)
	})

	return expl
}

func (expl *BlockExplorerServer) computeState() {
	longestChainHashList, err := expl.dag.GetLongestChainHashList(expl.dag.FullTip.Hash, expl.dag.FullTip.Height)
	if err != nil {
		expl.log.Fatalf("Failed to get longest chain hash list: %s", err)
	}

	// TODO 
	stateMachine, err := nakamoto.NewStateMachine(nil)
	if err != nil {
		expl.log.Fatalf("Failed to create state machine: %s", err)
	}
	
	state, err := nakamoto.RebuildState(expl.dag, *stateMachine, longestChainHashList)
	if err != nil {
		expl.log.Fatalf("Failed to rebuild state: %s", err)
	}
	expl.state = state
}



func (expl *BlockExplorerServer) Start() {
	expl.log.Println("Starting explorer server...")
	expl.computeState()
	listenAddr := fmt.Sprintf("%s:%d", expl.host, expl.port)
	expl.log.Printf("Listening on http://%s", listenAddr)

	err := http.ListenAndServe(listenAddr, expl.router)
	if err != nil {
		expl.log.Fatal("ListenAndServe: ", err)
	}
}

func (expl *BlockExplorerServer) homePage(w http.ResponseWriter, r *http.Request) {
	fullTip, err := expl.dag.GetLatestFullTip()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	tmpl := expl.getTemplates("templates/index.html", "templates/_base_layout.html")
	err = tmpl.ExecuteTemplate(w, "index.html", map[string]interface{}{
		"Title": "Home",
		"fullTip": fullTip,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (expl *BlockExplorerServer) getChain(w http.ResponseWriter, r *http.Request) {
	// vars := mux.Vars(r)

	// Get path param ?from_height
	// fromHeight := -1
	// if fromHeightStr, ok := vars["from_height"]; ok {
	// 	fromHeight, err := strconv.Atoi(fromHeightStr)
	// 	if err != nil {
	// 		http.Error(w, err.Error(), http.StatusInternalServerError)
	// 		return
	// 	}
	// }

	// Get the full chain hash list.
	chain, err := expl.dag.GetLongestChainHashList(expl.dag.FullTip.Hash, expl.dag.FullTip.Height + 1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// For each of those blocks, get the block details.
	blocks := make([]*nakamoto.Block, 0, len(chain))
	for i := 0; i < 25; i++ {
		blockheight := len(chain) - i - 1
		blockHash := chain[blockheight]
		block, err := expl.dag.GetBlockByHash(blockHash)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		blocks = append(blocks, block)
	}
	

	// Render.
	tmpl := expl.getTemplates("templates/chain.html", "templates/_base_layout.html")
	err = tmpl.ExecuteTemplate(w, "chain.html", map[string]interface{}{
		"Title": "Blockchain",
		"Chain": chain,
		"Blocks": blocks,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (expl *BlockExplorerServer) getBlock(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	blockHash_ := vars["id"]
	blockHash := nakamoto.HexStringToBytes32(blockHash_)

	// Lookup block
	block, err := expl.dag.GetBlockByHash(blockHash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Lookup transactions
	txs, err := expl.dag.GetBlockTransactions(blockHash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Render.
	tmpl := expl.getTemplates("templates/block.html", "templates/_base_layout.html")
	err = tmpl.ExecuteTemplate(w, "block.html", map[string]interface{}{
		"Title": fmt.Sprintf("Block #%d (%x)", block.Height, block.Hash),
		"Block": block,
		"Transactions": txs,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (expl *BlockExplorerServer) getEpoch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	epochId_ := vars["id"]

	// Lookup block
	epoch, err := expl.dag.GetEpochById(epochId_)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Render.
	tmpl := expl.getTemplates("templates/epoch.html", "templates/_base_layout.html")
	err = tmpl.ExecuteTemplate(w, "epoch.html", map[string]interface{}{
		"Title": fmt.Sprintf("Epoch %s", epochId_),
		"Epoch": epoch,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (expl *BlockExplorerServer) getAccount(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	accountPubkey_ := vars["id"]
	b, _ := hex.DecodeString(accountPubkey_)
	var fbuf [65]byte
	copy(fbuf[:], b)
	fmt.Printf("%v", expl.state)
	fmt.Printf("%v", fbuf)
	accountBalance := expl.state.GetBalance(fbuf)
	
	tmpl := expl.getTemplates("templates/account.html", "templates/_base_layout.html")
	err := tmpl.ExecuteTemplate(w, "account.html", map[string]interface{}{
		"Title": fmt.Sprintf("Account (%s)", accountPubkey_),
		"AccountPubkey": accountPubkey_,
		"AccountBalance": accountBalance,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}


func (expl *BlockExplorerServer) getAccounts(w http.ResponseWriter, r *http.Request) {
	type account struct {
		Pubkey [65]byte
		Balance uint64
	}
	accounts := make([]account, 0)
	for pubkey, balance := range expl.state.GetState() {
		accounts = append(accounts, account{
			Pubkey: pubkey,
			Balance: balance,
		})
	}

	// Render.
	tmpl := expl.getTemplates("templates/accounts.html", "templates/_base_layout.html")
	err := tmpl.ExecuteTemplate(w, "accounts.html", map[string]interface{}{
		"Title": "Ledger",
		"Accounts": accounts,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (expl *BlockExplorerServer) getTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txHash_ := vars["id"]
	txHash := nakamoto.HexStringToBytes32(txHash_)

	// Lookup transactions
	tx, err := expl.dag.GetTransactionByHash(txHash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	tmpl := expl.getTemplates("templates/transaction.html", "templates/_base_layout.html")
	err = tmpl.ExecuteTemplate(w, "transaction.html", map[string]interface{}{
		"Title": fmt.Sprintf("Transaction (%x)", tx.Hash),
		"Transaction": tx,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func NewLogger(prefix string, prefix2 string) *log.Logger {
	prefixFull := color.HiGreenString(fmt.Sprintf("[%s] ", prefix))
	if prefix2 != "" {
		prefixFull += color.HiYellowString(fmt.Sprintf("(%s) ", prefix2))
	}
	return log.New(os.Stdout, prefixFull, log.Ldate|log.Ltime|log.Lmsgprefix)
}