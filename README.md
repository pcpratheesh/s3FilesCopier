# s3FilesCopier
AWS lambda function to copy a file from s3 to remote server

# Deployment
 - Download the deployment file from https://github.com/pratheeshpcplpta/s3FilesZipper
 - Build the go with the following command  -   **env GOOS=linux go build -o main main.go && zip deployment.zip main**

 - Upload the deployment file to lambda
 - Set the configuration

 Handler function name: main
