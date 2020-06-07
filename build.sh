# build go binary
GOOS=linux go build -o main

# create zip for aws lambda
zip scheduler.zip main