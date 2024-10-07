go mod init ssh_server_go   # Inizializza il modulo Go, se non l'hai già fatto
go mod tidy                 # Aggiorna le dipendenze
go build -o ssh_server_go      # Compila il codice e crea un eseguibile chiamato `ssh_server`
./ssh_server_go
