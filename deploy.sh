echo "Start Deploying to AWS Lambda"
aws lambda update-function-code --function-name mfeMomentScheduler --zip fileb://scheduler.zip --region us-east-1
echo "Done Deploying to AWS Lambda"