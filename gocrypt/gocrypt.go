package gocrypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

var gocryptRegex, _ = regexp.Compile("(\\[gocrypt\\|[^\\]]*)\\]")

/**
 * Icecrypt uses AES 256 GCM to encrypt and authenticate strings
 */
func createRandomBytes(length int) []byte {
	random_bytes := make([]byte, length)
	rand.Read(random_bytes)
	return random_bytes
}

func CreateKey() []byte {
	return createRandomBytes(32)
}

func CreateIV() []byte {
	return createRandomBytes(12)
}

func createCipher(key []byte) (cipher.AEAD, error) {
	aes, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(aes)
	if err != nil {
		return nil, err
	}
	return aesgcm, nil
}

func encrypt(plaintext []byte, key []byte, iv []byte, ad []byte) ([]byte, error) {
	aesgcm, err := createCipher(key)
	if (err != nil) {
		return nil, err
	}
	return aesgcm.Seal(nil, iv, plaintext, ad), nil
}

func decrypt(ciphertext []byte, key []byte, iv []byte, ad []byte) ([]byte, error) {
	aesgcm, err := createCipher(key)
	if (err != nil) {
		return nil, err
	}

	return aesgcm.Open(nil, iv, ciphertext, ad)
}

func EncryptTags(buffer []byte, keyname string, key []byte) ([]byte, error) {

	if (!utf8.Valid(buffer)) {
		return nil, errors.New("File is not valid UTF-8")
	}

	match := gocryptRegex.Match(buffer)
	
	if (match) {
		matches := gocryptRegex.FindAllSubmatch(buffer, -1)
		for _, match := range matches {
			// The string we need is in the first capture group
			matchStr := string(match[1])
			parts := strings.Split(matchStr, "|")
			if (len(parts) > 3) {
				fmt.Printf("Block %s already encrypted.  Skipping.\n", match)	
			} else {
				iv := CreateIV()
				cipherText, err := encrypt([]byte(parts[1]), key, iv, []byte(parts[2]))
				if (err != nil) {
					return nil, err
				}

				replacement := fmt.Sprintf("[gocrypt|%s|%s|%s|%s]",
					base64.StdEncoding.EncodeToString(cipherText),
					keyname,
					base64.StdEncoding.EncodeToString(iv),
					parts[2])
				fmt.Println("Encrypted version:", replacement)

				buffer = bytes.Replace(buffer, match[0], []byte(replacement), 1)
			}

		}
	}

	return buffer, nil;
}

func DecryptTags(buffer []byte, keyroot string) ([]byte, error) {

	if (!utf8.Valid(buffer)) {
		return nil, errors.New("File is not valid UTF-8")
	}

	match := gocryptRegex.Match(buffer)

	if (match) {
		matches := gocryptRegex.FindAllSubmatch(buffer, -1)
		for _, match := range matches {
			// The string we need is in the first capture group
			matchStr := string(match[1])
			fmt.Println("match:", matchStr)
			parts := strings.Split(matchStr, "|")
			if (len(parts) < 5) {
				fmt.Printf("Block %s not correctly encrypted.  Skipping.\n", match)	
			} else {
				ct, err := base64.StdEncoding.DecodeString(parts[1])
				if (err != nil) {
					fmt.Println("Unable to decode ciphertext", parts[1], err)
					return nil, err
				}

				iv, err := base64.StdEncoding.DecodeString(parts[3])
				if (err != nil) {
					fmt.Println("Unable to decode IV", err)
					return nil, err
				}

				keyfile, err := ioutil.ReadFile(filepath.Join(keyroot, parts[2]))
				if (err != nil) {
					fmt.Println("Unable to read file for encryption", err)
					return nil, err
				}

				key, err := base64.StdEncoding.DecodeString(string(keyfile))
				if (err != nil) {
					fmt.Println("Unable to decode key", err)
					return nil, err
				}

				plainText, err := decrypt(ct, key, iv, []byte(parts[4]))
				if (err != nil) {
					return nil, err
				}

				buffer = bytes.Replace(buffer, match[0], []byte(plainText), 1)
			}
		}
	}

	return buffer, nil;
}
