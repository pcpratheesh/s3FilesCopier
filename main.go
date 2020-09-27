/**
 *
 * Author : <pratheesh@techversantinfo.com>
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

	AuthKeyBucket string `json:"bucket"`
	Authkey       string `json:"authkeypath"`

	FileCopyDestinationFolder string `json:"filedestinationfolder"`
}

var config Config

// load configuration from env
// configuration in format of base64 encode
func parseConfig() (err error) {
	data := make([]byte, 0)

	data, err = base64.StdEncoding.DecodeString(os.Getenv("CONFIG"))
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
		return fmt.Errorf("Connection configuration not found")
	}
	return nil
}

// Lambda execution callback
func Handler(event interface{}) error {

	//load configuration
	err := parseConfig()

	if err != nil {
		return fmt.Errorf("error in parsing the config %v", err)
	}

	//validate inputs
	err = validateConfig()

	e := event.(map[string]interface{})

	if e == nil {
		return fmt.Errorf("Unable to find payload")
	}

	payLoad := e["responsePayload"]
	payLoadMap := payLoad.(map[string]interface{})

	//create session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.Region)},
	)

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
			log.Fatalf("parse key failed:%v", err)
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

	fmt.Println("connected to ssh")

	// create new SFTP client
	client, err := sftp.NewClient(conn)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// create destination file
	dstFile, err := client.Create(config.FileCopyDestinationFolder + payLoadMap["filename"].(string))
	if err != nil {
		log.Fatalf("error creating destination %v ", err)
	}
	defer dstFile.Close()

	// create source file
	out, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(payLoadMap["destinationbucket"].(string)),
		Key:    aws.String(payLoadMap["filepath"].(string) + payLoadMap["filename"].(string)),
	})

	if err != nil {
		log.Fatalf("error creating source %v ", err)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(out.Body)

	srcFile := buf

	fmt.Println("read source file")

	bytes, err := io.Copy(dstFile, srcFile)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d bytes copied\n", bytes, srcFile)

	return nil
}

func main() {
	lambda.Start(Handler)
}
