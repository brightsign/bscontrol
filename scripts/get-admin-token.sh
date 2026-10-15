curl --location 'https://auth.bsn.cloud/realms/bsncloud/protocol/openid-connect/token' \
--header 'Content-Type: application/x-www-form-urlencoded' \
--header 'Authorization: Basic YnNuLXN1cHBvcnQ6ZXVudXZvbzNlaW04amFpbG9HOGhpZXBob29xdWl1eGU=' \
--data-urlencode 'grant_type=password' \
--data-urlencode 'username=$1 \
--data-urlencode 'password=$2
