# s3FilesCopier
AWS lambda function to copy a file from s3 to remote server

# Deployment
 - Download the deployment file from https://github.com/pratheeshpcplpta/s3FilesZipper
 - Build the go with the following command  -   **env GOOS=linux go build -o main main.go && zip deployment.zip main**

 - Upload the deployment file to lambda
 - Set the configuration

 Handler function name: main


# Configurations
You have to provide the configuration as base64 encoded format. Add **CONFIG** in lambda configuration variable

Use https://www.base64encode.org/ to encode the configurations

```json
{   	
 "region" : "Region",
 "user" : "Remote server access user",
 "password" : "Remote server access password - ",
 "host" : "Remote host",
 "port" : "Remote access port",
 "authkeybucket" : "Optional - If trying to access ssh with pem key : Pem key file stored bucket",
 "authkeypath" : "Optional - If trying to access ssh with pem key : pem key file path",
 "sourcebucket" : "File Source Bucket",
 "sourcefilepath" : "File path",
 "filedestinationfolder" :"Folder path where you want to copy the files at remote host"
}

```
