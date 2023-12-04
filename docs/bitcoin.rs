use crate::block::{Block, BlockHash};
pub mod block;

use std::fs;
use std::cmp;
use sha2::{Sha256, Digest};
use std::time::{SystemTime};
use ethereum_types::{U256};

const DICT_PATH: &str = "data/dict.txt";

// Returns the base-n representation of a number according to an
// alphabet specified in `symbols`. The length of symbols is the
// "base".
// 
// e.g. symbols = ["0","1"] is the binary numeral system.
// 
fn to_digits(num: u32, symbols: &Vec<&str>) -> String {
    let base = u32::try_from(symbols.len()).unwrap();
    let mut digits = String::new();
    let mut n = num;

    while n > 0 {
        let i = n.rem_euclid(base);
        if !digits.is_empty() {
            digits += " ";
        }
        digits += symbols[usize::try_from(i).unwrap()];
        n /= base;
    }

    digits
}

fn sha256_digest(content: &String) -> U256 {
    let mut hasher = Sha256::new();
    hasher.update(content.clone());
    let digest = hasher.finalize();
    return U256::from(digest.as_slice());
}

// fn current_time() -> u64 {
//     match SystemTime::now().duration_since(SystemTime::UNIX_EPOCH) {
//         Ok(n) => n.as_secs(),
//         Err(_) => panic!("SystemTime before UNIX EPOCH!"),
//     }
// }

fn solve_pow(target: U256, block: &Block, dict: &Vec<&str>) -> u32 {
    let mut nonce_idx: u32 = 0;
    let _base = dict.len();

    let mut block2 = block.clone();

    loop {
        block2.nonce = nonce_idx;

        // Build the outrage string.
        // Convert nonce number into alphabet of the outrage wordlist.
        // let outrage = to_digits(nonce_idx, &dict).as_bytes();

        // Compute SHA256 digest.
        let mut hasher = Sha256::new();
        let buf = serde_json::to_vec(&block2).unwrap();
        hasher.update(buf);
        let digest = hasher.finalize();

        // Convert to U256 number.
        let guess = U256::from(digest.as_slice());

        if guess < target {
            // println!("solved target={} value={} dist={} nonce=\"{}\"", target, guess, target - guess, outrage);
            // target = target / 2;
            return nonce_idx
        }

        nonce_idx += 1;
    }
}

fn main() {
    let word_dict = fs::read_to_string(DICT_PATH).expect("Unable to read file");
    let dict: Vec<&str> = word_dict.lines().collect();
    let NULL_HASH: U256 = U256::from(0);

    println!("Loaded dictionary of {} words", dict.len());
    
    // 
    // Mining loop.
    // 

    // This implements a proof-of-work algorithm, whereby the miner
    // searches for a value (`guess`) that is less than a `target`.
    // The lower the target, the higher the difficulty involved in the search process.
    // Unlike the usual PoW algorithms, the input content to the hash function is human-readable outrage propaganda. 
    // A nonce is generated, which is used to index into a dictionary of outrage content, and then thereby
    // hashed.
    let genesis_block = Block {
        prev_block_hash: NULL_HASH,
        number: 0,
        dict_hash: sha256_digest(&word_dict.to_string()),
        nonce: 0
    };
    let mut prev_block = genesis_block;
    let mut target: U256 = U256::from_dec_str("4567192616659071619386515177772132323232230222220222226193865124364247891968").unwrap();

    let EPOCH_LENGTH = 5;
    let EPOCH_TARGET_TIMESPAN_SECONDS = 5;
    let mut epoch_start_block_mined_at = SystemTime::now();


    println!("Starting miner...");
    println!("  difficulty = {}", target);
    println!();

    loop {
        let mut pending_block = Block {
            prev_block_hash: prev_block.block_hash(),
            number: prev_block.number + 1,
            dict_hash: sha256_digest(&word_dict.to_string()),
            nonce: 0
        };

        let guess = solve_pow(target, &pending_block, &dict);
        let outrage = to_digits(guess, &dict);
        println!("solved target={} value={} dist={} nonce=\"{}\"", target, guess, target - guess, outrage);

        // Seal block.
        pending_block.nonce = guess;
        pending_block.prev_block_hash = prev_block.block_hash();

        prev_block = pending_block;
        println!("mined block #{} hash={}\n", prev_block.number, prev_block.block_hash());

        // Update difficulty/target.
        if prev_block.number % EPOCH_LENGTH == 0 {
            // 5 seconds for epoch of 5 blocks
            
            let mut timespan = SystemTime::now().duration_since(epoch_start_block_mined_at).unwrap().as_secs();
            if timespan < EPOCH_TARGET_TIMESPAN_SECONDS/4 {
                timespan = EPOCH_TARGET_TIMESPAN_SECONDS/4;
            }
            if timespan > EPOCH_TARGET_TIMESPAN_SECONDS*4 {
                timespan = EPOCH_TARGET_TIMESPAN_SECONDS*4;
            }
            
            let epoch = prev_block.number / EPOCH_LENGTH;
            let factor = timespan / EPOCH_TARGET_TIMESPAN_SECONDS;
            target = target * timespan / EPOCH_TARGET_TIMESPAN_SECONDS;

            println!("epoch #{}", epoch);
            println!("adjusting difficulty timespan={}s factor={}", timespan, (timespan as f64) / (EPOCH_TARGET_TIMESPAN_SECONDS as f64));
            
            epoch_start_block_mined_at = SystemTime::now();
        }
    }
}
