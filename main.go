/**
 *
 * Author : <pratheesh>
 *
 * AWS Lambda function
 *
 * To copy s3 bucket content to a remote server
 */

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/pkg/sftp"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type Config struct {
	Region string `json:"region"`

	User     string `json:"user"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     string `json:"port"`

	AuthKeyBucket string `json:"authkeybucket"`
	Authkey       string `json:"authkeypath"`

	FileName                  string `json:"filename"`
	FilePath                  string `json:"sourcefilepath"`
	SourceFileBucket          string `json:"sourcebucketbucket"`
	FileCopyDestinationFolder string `json:"filedestinationfolder"`
}

var config Config

// load configuration from env
// configuration in format of base64 encode
func parseConfig() (err error) {
	data, err := base64.StdEncoding.DecodeString(os.Getenv("CONFIG"))
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("no configuration available")
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return err
	}
	return nil
}

// validate configs
func validateConfig() error {
	if config.Authkey == "" && config.AuthKeyBucket == "" && config.Password == "" {
		return fmt.Errorf("connection configuration not found")
	}
	return nil
}

// Lambda execution callback
func Handler(event interface{}) error {

	// initiate logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	//load configuration
	err := parseConfig()

	if err != nil {
		return fmt.Errorf("error in parsing the config %v", err)
	}

	//validate inputs
	err = validateConfig()
	if err != nil {
		logger.Error("unable to validate configuration", zap.Error(err))
		return err
	}
	e := event.(map[string]interface{})

	if e == nil {
		logger.Error("unable to find payload", zap.Error(err))
		return fmt.Errorf("unable to find payload")
	}

	//create session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.Region)},
	)
	if err != nil {
		logger.Error("unable initiate aws connection", zap.Error(err))
		return err
	}
	// initialize
	svc := s3.New(sess)

	var Authentications []ssh.AuthMethod

	//check the Authkey exists
	if config.Authkey != "" {
		// read pem key file from s3
		out, err := svc.GetObject(&s3.GetObjectInput{
			Bucket: aws.String(config.AuthKeyBucket),
			Key:    aws.String(config.Authkey),
		})

		if err != nil {
			log.Fatal(err)
		}
		buf := new(bytes.Buffer)
		buf.ReadFrom(out.Body)

		pemBytes := buf.Bytes()
		if err != nil {
			log.Fatal(err)
		}

		signer, err := ssh.ParsePrivateKey(pemBytes)
		if err != nil {
			logger.Error("parse key failed:%v", zap.Error(err))
			return err
		}

		Authentications = append(Authentications, ssh.PublicKeys(signer))
	} else {
		Authentications = append(Authentications, ssh.Password(config.Password))
	}

	clientConfig := &ssh.ClientConfig{
		User:            config.User,
		Auth:            Authentications,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		// HostKeyCallback: ssh.FixedHostKey(hostKey),
	}

	// connect to Remote system
	conn, err := ssh.Dial("tcp", config.Host+":"+config.Port, clientConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	logger.Info("connected to ssh")

	// create new SFTP client
	client, err := sftp.NewClient(conn)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// create destination file
	dstFile, err := client.Create(config.FileCopyDestinationFolder + config.FileName)
	if err != nil {
		log.Fatalf("error creating destination %v ", err)
	}
	defer dstFile.Close()

	// create source file
	out, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(config.SourceFileBucket),
		Key:    aws.String(config.FilePath + config.FileName),
	})

	if err != nil {
		logger.Error("error creating source %v ", zap.Error(err))
		return err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(out.Body)

	srcFile := buf

	logger.Info("read source file")

	bytes, err := io.Copy(dstFile, srcFile)
	if err != nil {
		log.Fatal(err)
	}
	logger.Info("%d bytes copied\n", zap.Any("", bytes), zap.Any("", srcFile))

	return nil
}

func main() {
	lambda.Start(Handler)
}
