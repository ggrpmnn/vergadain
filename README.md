# Vergadain

Vergadain is a small tool written in Go to describe JIRA customfields and any specific values that tie to those fields. This tool will list any fields that the provided user has access to view.

The user credentials and JIRA instance target are provided on the command line. Input into the password prompt is hidden (to protect your credentials).

### Requirements

1. A JIRA instance
2. An account (username and password) on that JIRA instance
3. Permissions to see at least one project

### Build & Usage

You can use `go run main.go` or `go build -o vergadain; ./vergadain` to run the tool. To specify an output file, use the `-f <filepath>` option.

### Credentials File

When run, the tool will ask for the user's credentials in the terminal. As this makes searching for specific fields slow, it also is able to take in user credentials via a JSON file. The file should have the format specified below:
```
{
    "username": "<your username>",
    "password": "<your password>",
    "site_url": "https://your.jira.instance.xyz/"
}
