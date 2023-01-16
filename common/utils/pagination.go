package utils

import (
	"bytes"
	"crypto/aes"
	"encoding/base64"
	"fmt"
	"github.com/TakeoffTech/site-info-svc/common"
	"go.uber.org/zap"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// GetNextPageToken will take data as string input and a string key of length 22
// For getting the next page token we are using AES encryption algorithm
// First we take the key of length 22 then we prefix it with the current timestamp
// So the final encryption key = timestamp(length 10) + key (length 22)
// Using this key we encrypt the data.
// Then we return the base 64 encoded value of "timestamp::::encryptedValue" as the next_page_token
func GetNextPageToken(data string, key string) (string, error) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	key = timestamp + key
	cipher, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	var array []byte
	// Calculate the number of blocks for the data that is sent for encryption
	// according to the AES Block size AES Block size=16
	// if data consists of 54 elements then block count would be ceil(54/16) = 4
	blockCount := math.Ceil(float64(len(data)) / float64(cipher.BlockSize()))
	//Create a byte array for encryption
	//if size of data is less than block size then we create an array of 1 block size and copy data into it
	//remaining values would be null
	if len(data) < cipher.BlockSize() {
		array = make([]byte, cipher.BlockSize())
		copy(array, data)
	} else { //if data is greater than 1 block size then create the array of block count*block size
		array = make([]byte, cipher.BlockSize()*int(blockCount))
		copy(array, data)
	}
	//Create out array to store the encrypted data
	out := make([]byte, len(array))
	//Encrypt the data block by block and store in the out array
	for i := 0; i < int(blockCount); i++ {
		blockStart := i * cipher.BlockSize()
		blockEnd := blockStart + cipher.BlockSize()
		cipher.Encrypt(out[blockStart:blockEnd], array[blockStart:blockEnd])
	}

	//Prefix the encrypted data with timestamp separated by ColonSeparator
	token := timestamp + common.ColonSeparator + string(out)

	return base64.StdEncoding.EncodeToString([]byte(token)), nil
}

// DecodeNextPageToken will take the encrypted page_token and decode the data from it
// First we do a base64 decode and get the base64 decoded token
// decodedToken = "timestamp::::encryptedValue"
// Then we separate the timestamp and the encryptedValue
// Using the timestamp we create the key for decryption
// key used for decrypt = "timestamp+key" -> timestamp (length 10) + key (length 22)
// Note : as AES is symmetric key algo you have to use the same key that was used for encryption to decrypt
// using this decrypt key we decrypt the encryptedValue and return it as a string
func DecodeNextPageToken(token string, key string) (string, error) {
	decodedToken, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return "", err
	}
	decodedTokens := strings.Split(string(decodedToken), common.ColonSeparator)
	cipher, err := aes.NewCipher([]byte(decodedTokens[0] + key))
	if err != nil {
		return "", err
	}

	data := make([]byte, len(decodedTokens[1]))
	//Using the length of encrypted data find out the block count
	blockCount := math.Ceil(float64(len(data)) / float64(cipher.BlockSize()))
	// Decrypt the encrypted data decodedTokens[1] block by block and store in the data array
	for i := 0; i < int(blockCount); i++ {
		blockStart := i * cipher.BlockSize()
		blockEnd := blockStart + cipher.BlockSize()
		cipher.Decrypt(data[blockStart:blockEnd], []byte(decodedTokens[1])[blockStart:blockEnd])
	}

	// trim the last block if it contains null elements in the byte array
	return string(bytes.Trim(data, "\x00")), nil
}

// ValidatePageToken will validate the page_token value of the header passed into the request
// It will check if the value can be base64 decoded without any issues
// also validate if the token is in correct format
// also validate the expiration of the token
func ValidatePageToken(request *http.Request, header string) []string {
	var errors []string
	decodedToken, err := base64.StdEncoding.DecodeString(request.Header.Get(header))
	if err != nil {
		errors = append(errors, fmt.Sprintf("Invalid header value, unable to decrypt header : %v", header))
	} else {
		decodedTokens := strings.Split(string(decodedToken), common.ColonSeparator)
		if len(decodedTokens) != 2 || len(decodedTokens[0]) != 10 || len([]byte(decodedTokens[1])) < aes.BlockSize {
			errors = append(errors, fmt.Sprintf("Invalid %s header : %v", header, request.Header.Get(header)))
		} else if timestamp, err := strconv.ParseInt(decodedTokens[0], 10, 64); err != nil {
			errors = append(errors, fmt.Sprintf("Invalid %s : %v", header, request.Header.Get(header)))
		} else if time.Since(time.Unix(timestamp, 0)).Minutes() < 0 ||
			time.Since(time.Unix(timestamp, 0)).Minutes() > common.ExpireTokenDuration {
			errors = append(errors, fmt.Sprintf("Header %s expired : %v", header, request.Header.Get(header)))
		}
	}

	return errors
}

// AddPaginationHeaderIfNotAdded will add required header as page_token and page_size if not added in request.
func AddPaginationHeaderIfNotAdded(request *http.Request) []string {
	var headers []string
	if request.Header.Get(common.HeaderPageToken) != "" {
		headers = append(headers, common.HeaderPageToken)
	}
	if request.Header.Get(common.HeaderPageSize) != "" {
		headers = append(headers, common.HeaderPageSize)
	}

	return headers
}

// GetPageSizeFromHeader will get page size from header `page_size` if added in request, else will be default.
// in case of any error, pageSize will be return -1
func GetPageSizeFromHeader(request *http.Request, logger *zap.SugaredLogger) int {
	pageSize := common.DefaultPageSize
	pageSizeStr := request.Header.Get(common.HeaderPageSize)
	if pageSizeStr != "" {
		var err error
		pageSize, err = strconv.Atoi(pageSizeStr)
		if err != nil {
			logger.Errorf("Received page_size as %s , err is %v", pageSizeStr, err)

			return common.ReturnError
		}
		logger.Debugf("Received page_size %d", pageSize)
	} else {
		logger.Debugf("default page_size %d considered", common.DefaultPageSize)
	}

	return pageSize
}
