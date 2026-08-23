package wallet

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/scrypt"
)

const (
	scryptN      = 32768
	scryptR      = 8
	scryptP      = 1
	scryptKeyLen = 32

	walletSaltSize = 16
)

type encryptedWalletFile struct {
	Version    uint32 `json:"version"`
	KDF        string `json:"kdf"`
	Cipher     string `json:"cipher"`
	Salt       string `json:"salt"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

// SaveEncrypted saves a Sudharma Network wallet encrypted
// using a password-derived AES-256-GCM key.
func (w *Wallet) SaveEncrypted(
	path string,
	password string,
) error {

	if w == nil {
		return fmt.Errorf(
			"wallet cannot be nil",
		)
	}

	if w.PrivateKey == nil {
		return fmt.Errorf(
			"wallet private key cannot be nil",
		)
	}

	if path == "" {
		return fmt.Errorf(
			"wallet path cannot be empty",
		)
	}

	if len(password) < 12 {
		return fmt.Errorf(
			"wallet password must contain at least 12 characters",
		)
	}

	salt :=
		make(
			[]byte,
			walletSaltSize,
		)

	if _, err :=
		rand.Read(salt); err != nil {

		return fmt.Errorf(
			"failed to generate wallet salt: %w",
			err,
		)
	}

	key, err :=
		scrypt.Key(
			[]byte(password),
			salt,
			scryptN,
			scryptR,
			scryptP,
			scryptKeyLen,
		)

	if err != nil {
		return fmt.Errorf(
			"failed to derive encryption key: %w",
			err,
		)
	}

	block, err :=
		aes.NewCipher(key)

	if err != nil {
		return fmt.Errorf(
			"failed to create cipher: %w",
			err,
		)
	}

	gcm, err :=
		cipher.NewGCM(block)

	if err != nil {
		return fmt.Errorf(
			"failed to create AES-GCM: %w",
			err,
		)
	}

	nonce :=
		make(
			[]byte,
			gcm.NonceSize(),
		)

	if _, err :=
		rand.Read(nonce); err != nil {

		return fmt.Errorf(
			"failed to generate encryption nonce: %w",
			err,
		)
	}

	privateKeyBytes :=
		w.PrivateKey.D.Bytes()

	ciphertext :=
		gcm.Seal(
			nil,
			nonce,
			privateKeyBytes,
			nil,
		)

	fileData :=
		encryptedWalletFile{
			Version: 1,
			KDF:     "scrypt",
			Cipher:  "aes-256-gcm",

			Salt: hex.EncodeToString(
				salt,
			),

			Nonce: hex.EncodeToString(
				nonce,
			),

			Ciphertext: hex.EncodeToString(
				ciphertext,
			),
		}

	encoded, err :=
		json.MarshalIndent(
			fileData,
			"",
			"  ",
		)

	if err != nil {
		return fmt.Errorf(
			"failed to encode encrypted wallet: %w",
			err,
		)
	}

	directory :=
		filepath.Dir(path)

	if directory != "." {
		if err :=
			os.MkdirAll(
				directory,
				0700,
			); err != nil {

			return fmt.Errorf(
				"failed to create wallet directory: %w",
				err,
			)
		}
	}

	if err :=
		os.WriteFile(
			path,
			encoded,
			0600,
		); err != nil {

		return fmt.Errorf(
			"failed to save encrypted wallet: %w",
			err,
		)
	}

	return nil
}

// LoadEncrypted decrypts and restores a Sudharma Network wallet.
func LoadEncrypted(
	path string,
	password string,
) (*Wallet, error) {

	if path == "" {
		return nil,
			fmt.Errorf(
				"wallet path cannot be empty",
			)
	}

	if password == "" {
		return nil,
			fmt.Errorf(
				"wallet password cannot be empty",
			)
	}

	data, err :=
		os.ReadFile(path)

	if err != nil {
		return nil,
			fmt.Errorf(
				"failed to read encrypted wallet: %w",
				err,
			)
	}

	var stored encryptedWalletFile

	if err :=
		json.Unmarshal(
			data,
			&stored,
		); err != nil {

		return nil,
			fmt.Errorf(
				"invalid encrypted wallet file: %w",
				err,
			)
	}

	if stored.Version != 1 {
		return nil,
			fmt.Errorf(
				"unsupported wallet version: %d",
				stored.Version,
			)
	}

	if stored.KDF != "scrypt" {
		return nil,
			fmt.Errorf(
				"unsupported wallet KDF",
			)
	}

	if stored.Cipher != "aes-256-gcm" {
		return nil,
			fmt.Errorf(
				"unsupported wallet cipher",
			)
	}

	salt, err :=
		hex.DecodeString(
			stored.Salt,
		)

	if err != nil ||
		len(salt) != walletSaltSize {

		return nil,
			fmt.Errorf(
				"invalid wallet salt",
			)
	}

	nonce, err :=
		hex.DecodeString(
			stored.Nonce,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"invalid wallet nonce",
			)
	}

	ciphertext, err :=
		hex.DecodeString(
			stored.Ciphertext,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"invalid wallet ciphertext",
			)
	}

	key, err :=
		scrypt.Key(
			[]byte(password),
			salt,
			scryptN,
			scryptR,
			scryptP,
			scryptKeyLen,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"failed to derive decryption key: %w",
				err,
			)
	}

	block, err :=
		aes.NewCipher(key)

	if err != nil {
		return nil,
			fmt.Errorf(
				"failed to create cipher: %w",
				err,
			)
	}

	gcm, err :=
		cipher.NewGCM(block)

	if err != nil {
		return nil,
			fmt.Errorf(
				"failed to create AES-GCM: %w",
				err,
			)
	}

	if len(nonce) != gcm.NonceSize() {
		return nil,
			fmt.Errorf(
				"invalid wallet nonce size",
			)
	}

	privateKeyBytes, err :=
		gcm.Open(
			nil,
			nonce,
			ciphertext,
			nil,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"incorrect password or corrupted wallet",
			)
	}

	return walletFromPrivateKeyBytes(
		privateKeyBytes,
	)
}
