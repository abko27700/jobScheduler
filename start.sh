
go mod init daria.com/jobScheduler
go mod tidy
go build
nohup ./jobScheduler &

# Exit the script
exit 0