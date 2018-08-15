package classes

import (
	"fmt"
	"github.com/amstee/blockchain/models"
	"time"
	"github.com/jinzhu/gorm"
)

type Blockchain struct {
	blocks []*models.BlockModel
	db *gorm.DB
}

func (b* Blockchain) MineBlock(transactions []*models.TransactionModel) {
	b.AddBlock(transactions)
}

func (b* Blockchain) FindOutputs(PubKeyHash []byte, amount int) (int, map[string][]int) {
	outputs := make(map[string][]int)
	unspents := b.GetUnspentTransactions(PubKeyHash)
	total := 0

	for _, tx := range unspents {
		for count, out := range tx.Vout {
			if out.CanBeUnlocked(PubKeyHash) {
				total += out.Value
				outputs[out.TxID] = append(outputs[out.TxID], count)
				if total >= amount {
					return total, outputs
				}
			}
		}
	}
	return total, outputs
}

func (b* Blockchain) GetUnspentOutputs(PubKeyHash []byte) []models.TXOutput {
	var outputs []models.TXOutput
	txs := b.GetUnspentTransactions(PubKeyHash)

	for _, tx := range txs {
		for _, out := range tx.Vout {
			if out.CanBeUnlocked(PubKeyHash) {
				outputs = append(outputs, out)
			}
		}
	}
	return outputs
}

func (b *Blockchain) GetUnspentTransactions(PubKeyHash []byte) []models.TransactionModel {
	var unspent []models.TransactionModel
	spent := make(map[string] []int)
	var transactions []*models.TransactionModel
	it := len(b.blocks) - 1

	for it >= 0 {
		b.db.Model(&b.blocks[it]).Related(&transactions, "BlockID")
		it -= 1
		for _, tx := range transactions {
			b.db.Model(&tx).Related(&tx.Vin, "TxID")
			b.db.Model(&tx).Related(&tx.Vout, "TxID")
		NextIteration:
			for i, out := range tx.Vout {
				if spent[tx.Txid] != nil {
					for _, spentOut := range spent[tx.Txid] {
						if spentOut == i {
							continue NextIteration
						}
					}
				}
				if out.CanBeUnlocked(PubKeyHash) {
					unspent = append(unspent, *tx)
				}
			}
			if tx.IsCoinbase() == false {
				for _, in := range tx.Vin {
					if in.CanUnlockOutput(PubKeyHash) {
						spent[in.OtxID] = append(spent[in.OtxID], in.Vout)
					}
				}
			}
		}
	}
	return unspent
}


func (b *Blockchain) AddBlock(txs []*models.TransactionModel) {
	prevBlock := b.blocks[len(b.blocks) - 1]
	newBlock := NewBlock(txs, prevBlock)
	b.blocks = append(b.blocks, newBlock)
	if b.db.NewRecord(newBlock) {
		b.db.Create(newBlock)
	}
}

func NewGenesisBlock(db *gorm.DB, address string) *models.BlockModel {
	coinbase := NewCoinBaseTX(address, "")
	block := NewBlock([]*models.TransactionModel{coinbase}, nil)
	if db.NewRecord(block) {
		db.Create(block)
	}
	return block
}

func GetBlockChainFromGenesis(db *gorm.DB) *Blockchain {
	var blocks []*models.BlockModel

	if err := db.Find(&blocks).Error; err != nil || len(blocks) == 0 {
		return nil
	}
	return &Blockchain{blocks, db}
}

func NewBlockChain(db *gorm.DB, address string) *Blockchain {
	return &Blockchain{[]*models.BlockModel{NewGenesisBlock(db, address)}, db}
}

func (b *Blockchain) DisplayBlockChain() {
	var txs []*models.TransactionModel
	var inputs []models.TXInput
	var outputs []models.TXOutput
	for _, block := range b.blocks {
		fmt.Printf("PrevHash: %x\n", block.GetPrevHash())
		fmt.Printf("Hash: %x\n", block.GetHash())
		b.db.Model(&block).Related(&txs, "BlockID")
		for _, tx := range txs {
			fmt.Printf("Transaction BLOCK   : %d\n", tx.BlockID)
			fmt.Printf("Transaction ID      : %x\n", tx.GetTXID())
			b.db.Model(&tx).Related(&inputs, "TxID")
			b.db.Model(&tx).Related(&outputs, "TxID")
			for _, itx := range inputs {
				fmt.Printf("Input TXID          : %x\n", itx.GetTXID())
				fmt.Printf("Input OTXID         : %x\n", itx.GetOTXID())
				fmt.Printf("Input VOUT          : %x\n", itx.Vout)
				fmt.Printf("Input Signature     : %s\n", itx.Signature)
				fmt.Printf("Input PubKey        : %x\n", itx.PubKey)
				fmt.Printf("Input PubKeyHashed  : %x\n", itx.GetPubKeyHashed())
			}
			for _, otx := range outputs {
				fmt.Printf("Output TXID         : %x\n", otx.GetTXID())
				fmt.Printf("Output Value        : %d\n", otx.Value)
				fmt.Printf("Output PubKeyHashed : %x\n", otx.GetKey())
			}
		}
		fmt.Println()
	}
}

func NewBlock(txs []*models.TransactionModel, prev *models.BlockModel) *models.BlockModel {
	var id uint
	var prevHash string

	if prev == nil {
		id = 0
		prevHash = ""
	} else {
		id = prev.ID
		prevHash = prev.Hash
	}
	block := &models.BlockModel{
		Timestamp: time.Now().Unix(),
		Transactions: txs,
		Hash: "",
		Prev: id,
		PrevHash: prevHash,
		Iterations: 0,
	}
	pow := NewProofOfWork(block)
	i, hash := pow.Run()
	block.Hash = string(hash[:])
	block.Iterations = i
	return block
}