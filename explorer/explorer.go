package explorer

import (
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/gorilla/mux"
	"github.com/liamzebedee/tinychain-go/core/nakamoto"
)

//go:embed templates/*.html assets/*
var embedFS embed.FS

type BlockExplorerServer struct {
	router *mux.Router
	log    *log.Logger

	host        string
	port        int
	environment string

	dag   *nakamoto.BlockDAG
	state *nakamoto.StateMachine
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
		"dev":  localDirFS{baseDir: baseDir},
		"test": embedFS,
		"live": embedFS,
	}[expl.environment]

	return &fs
}


func timeAgo(ts uint64) string {
	t := time.UnixMilli(int64(ts))
	duration := time.Since(t)
	switch {
	case duration < time.Minute:
		return fmt.Sprintf("%d seconds ago", int(duration.Seconds()))
	case duration < time.Hour:
		return fmt.Sprintf("%d minutes ago", int(duration.Minutes()))
	case duration < time.Hour*24:
		return fmt.Sprintf("%d hours ago", int(duration.Hours()))
	default:
		return fmt.Sprintf("%d days ago", int(duration.Hours()/24))
	}
}

func formatTimestamp(ts uint64) string {
	t := time.UnixMilli(int64(ts))
	return t.Format("02 Jan 2006 15:04:05 MST")
}

func formatTimestampDatetime(ts uint64) string {
	t := time.UnixMilli(int64(ts))
	return t.Format("2006-01-02 15:04:05")
}


func (expl *BlockExplorerServer) getTemplates(patterns ...string) *template.Template {
	fs := *expl.getFS()
	funcMap := template.FuncMap{
		"timeAgo": timeAgo,
		"formatTimestamp": formatTimestamp,
		"formatTimestampDatetime": formatTimestampDatetime,
	}
	return template.Must(template.New("").Funcs(funcMap).ParseFS(fs, patterns...))
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
		"dev":  "127.0.0.1",
		"test": "0.0.0.0",
		"live": "0.0.0.0",
	}[environment]

	router := mux.NewRouter()

	expl := &BlockExplorerServer{
		router:      router,
		log:         log,
		host:        host,
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
	expl.router.HandleFunc("/search/", expl.search)

	// Serve static files.
	expl.router.PathPrefix("/assets/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	listenAddr := fmt.Sprintf("%s:%d", expl.host, expl.port)
	expl.log.Printf("Listening on http://%s", listenAddr)

	// recompute state on a loop.
	expl.computeState()
	go func() {
		latestFullTip := [32]byte{}

		for {
			time.Sleep(1 * time.Second)
			latestTip, err := expl.dag.GetLatestFullTip()
			if err != nil {
				expl.log.Fatalf("Failed to get latest tip: %s", err)
			}

			if latestTip.Hash != latestFullTip {
				expl.log.Println("Recomputing state...")
				expl.dag.UpdateTip()
				expl.computeState()
				expl.log.Println("Recomputing state done.")
				latestFullTip = expl.dag.FullTip.Hash
			}
		}
	}()

	err := http.ListenAndServe(listenAddr, expl.router)
	if err != nil {
		expl.log.Fatal("ListenAndServe: ", err)
	}
}

func (expl *BlockExplorerServer) search(w http.ResponseWriter, r *http.Request) {
	// Get the 'q' query parameter.
	query := r.URL.Query().Get("q")
	// Trim it.
	query = strings.Trim(query, " ")
	if query == "" {
		http.Error(w, "No search query provided", http.StatusBadRequest)
		return
	}

	// If q is 65 bytes, it's a pubkey.
	if len(query) == 65*2 {
		// Check if it's a pubkey.
		_, err := hex.DecodeString(query)
		if err != nil {
			http.Error(w, "Invalid pubkey", http.StatusBadRequest)
			return
		}

		// Redirect to account page.
		http.Redirect(w, r, fmt.Sprintf("/accounts/%s", query), http.StatusFound)
		return
	}

	// Else it could be a tx hash or a block hash.
	// Looking up the block is cheaper, because the blocks table grows slower than transactions table.
	// So we'll check if it's a block hash first.
	blockHash := nakamoto.HexStringToBytes32(query)
	block, err := expl.dag.GetBlockByHash(blockHash)
	if err != nil && !errors.Is(err, nakamoto.ErrBlockNotFound) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if block != nil {
		http.Redirect(w, r, fmt.Sprintf("/blocks/%s", query), http.StatusFound)
		return
	}

	// If it's not a block hash, it could be a transaction hash.
	tx, err := expl.dag.GetTransactionByHash(blockHash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tx != nil {
		http.Redirect(w, r, fmt.Sprintf("/transactions/%s", query), http.StatusFound)
		return
	}

	// Render a no results found page.
	tmpl := expl.getTemplates("templates/search.html", "templates/_base_layout.html")
	err = tmpl.ExecuteTemplate(w, "search.html", map[string]interface{}{
		"Title": "Search",
		"Query": query,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	return
}

func (expl *BlockExplorerServer) homePage(w http.ResponseWriter, r *http.Request) {
	fullTip, err := expl.dag.GetLatestFullTip()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl := expl.getTemplates("templates/index.html", "templates/_base_layout.html")
	err = tmpl.ExecuteTemplate(w, "index.html", map[string]interface{}{
		"Title":   "Home",
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
	chain, err := expl.dag.GetLongestChainHashList(expl.dag.FullTip.Hash, expl.dag.FullTip.Height+1)
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
		"Title":  "Blockchain",
		"Chain":  chain,
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
		"Title":        fmt.Sprintf("Block #%d (%x)", block.Height, block.Hash),
		"Block":        block,
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
	accountPubkey := fbuf
	accountBalance := expl.state.GetBalance(fbuf)

	// Get all the transactions for an account.
	db := expl.dag.GetDB()
	rows, err := db.Query("SELECT hash, sig, from_pubkey, to_pubkey, amount, fee, nonce, version FROM transactions WHERE from_pubkey = ? OR to_pubkey = ?", accountPubkey[:], accountPubkey[:])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	transactions := make([]nakamoto.Transaction, 0)
	for rows.Next() {
		tx := nakamoto.Transaction{}
		hash := []byte{}
		sig := []byte{}
		fromPubkey := []byte{}
		toPubkey := []byte{}
		amount := uint64(0)
		fee := uint64(0)
		nonce := uint64(0)
		version := 0 // TODO

		err := rows.Scan(&hash, &sig, &fromPubkey, &toPubkey, &amount, &fee, &nonce, &version)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		copy(tx.Hash[:], hash)
		copy(tx.Sig[:], sig)
		copy(tx.FromPubkey[:], fromPubkey)
		copy(tx.ToPubkey[:], toPubkey)
		tx.Amount = amount
		tx.Fee = fee
		tx.Nonce = nonce
		tx.Version = byte(version)

		transactions = append(transactions, tx)
	}

	tmpl := expl.getTemplates("templates/account.html", "templates/_base_layout.html")
	err = tmpl.ExecuteTemplate(w, "account.html", map[string]interface{}{
		"Title":          fmt.Sprintf("Account (%s)", accountPubkey_),
		"AccountPubkey":  accountPubkey_,
		"AccountBalance": accountBalance,
		"Transactions":   transactions,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (expl *BlockExplorerServer) getAccounts(w http.ResponseWriter, r *http.Request) {
	type account struct {
		Pubkey  [65]byte
		Balance uint64
	}
	accounts := make([]account, 0)
	for pubkey, balance := range expl.state.GetState() {
		accounts = append(accounts, account{
			Pubkey:  pubkey,
			Balance: balance,
		})
	}

	// Now sort the accounts by balance desc.
	slices.SortFunc(accounts, func(a, b account) int {
		if a.Balance > b.Balance {
			return -1
		}
		if a.Balance < b.Balance {
			return 1
		}
		return 0
	})

	// Render.
	tmpl := expl.getTemplates("templates/accounts.html", "templates/_base_layout.html")
	err := tmpl.ExecuteTemplate(w, "accounts.html", map[string]interface{}{
		"Title":    "Ledger",
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

	// Get the blocks the transaction is included in.
	blocks, err := expl.dag.GetTransactionBlocks(txHash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Compute the tx status.
	type ConfirmationInfo struct {
		Status string
		Block  nakamoto.Block
	}

	txStatus := ConfirmationInfo{
		Status: "Unconfirmed",
	}
	// confirmed := make(map[[32]byte]bool)
	chain, err := expl.dag.GetLongestChainHashList(expl.dag.FullTip.Hash, expl.dag.FullTip.Height+1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, confirmedBlock := range chain {
		// Check if blocks array contains this block.
		for _, block := range blocks {
			if block == confirmedBlock {
				// confirmed[block] = true
				txStatus.Status = "Confirmed"
				block_, err := expl.dag.GetBlockByHash(confirmedBlock)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				txStatus.Block = *block_
			}
		}
	}

	// Render.
	tmpl := expl.getTemplates("templates/transaction.html", "templates/_base_layout.html")
	err = tmpl.ExecuteTemplate(w, "transaction.html", map[string]interface{}{
		"Title":       fmt.Sprintf("Transaction (%x)", tx.Hash),
		"Transaction": tx,
		"Blocks":      blocks,
		"TxStatus":    txStatus,
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
