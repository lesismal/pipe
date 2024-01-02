package packer

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
)

type AESCBC struct {
	Key, IV []byte
}

func (packer *AESCBC) Pack(originData []byte) ([]byte, error) {
	block, err := aes.NewCipher(packer.Key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	originData = packer.pkcs7Padding(originData, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, packer.IV[:blockSize])
	crypted := make([]byte, len(originData))
	blockMode.CryptBlocks(crypted, originData)
	return crypted, nil
}

func (packer *AESCBC) Unpack(crypted []byte) ([]byte, error) {
	block, err := aes.NewCipher(packer.Key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, packer.IV[:blockSize])
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	origData = packer.pkcs7Unpadding(origData)
	return origData, nil
}

func (packer *AESCBC) pkcs7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func (packer *AESCBC) pkcs7Unpadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}
