# opexec
a 1Password binary for OpenClaw SecretRef

## Background
There are lots of ways on the internet to use 1Password with OpenClawy, but a lot of them didn't fit what I really wanted it to do. Most of them require creating one "secret provider" for each secret you want to use in your openclaw.json config and that just seemed awkward to me. The exec provider must be a standalone binary (which I didn't realize until I built one in python) so I decided to learn golang and make this.

## Buildling and Installing
This is my first golang project, so I'm sure there are more elegant ways to set this up. If you know golang but are new to OpenClaw, it's important to remember that it runs in user space; it's not meant to be installed for all users. Hence, my ideas about how this can work are based on that

What I did on my installation (debian-based headless):

* clone the repo to a folder in my home directory
* go into the opexec folder
* run this command `go build -o ~/.local/bin/opexec`

## Appendix
I developed this on my mac with Visual Studio Code. If you'd like to do the same, first look in the `payloads` folder and make a copy of `example_request_payload.json` called `payload.json`. Then you have to go into that file and change the string from `op://Vault/Item/field` to something that's valid for your 1Password.

After you do that, below is an example `.vscode/launch.json` to start from. Make sure to insert your 1Password service account token.
```
{
    "version": "0.2.0",
    "configurations": [

        {
            "name": "Launch Package",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "opexec.go",
            "stdinFrom": "${workspaceFolder}/payloads/payload.json",
            "env": {
                "OP_SERVICE_ACCOUNT_TOKEN": "<your_token_here>"
            }
        }
    ]
}
```

#### About 1Password
This is intended for people that are already using 1Password. You can look at https://www.1password.dev/get-started$0 for a lot of information, but here's what I'd recommend to get started.

* Create a new Vault for the secrets for OpenClaw.
* Create a Service Account https://www.1password.dev/service-accounts/get-started$0
* Authorize the Service Account to have read access to the new Vault. (For this use, you only need read access to only that one Vault.)
* Save the API Key for the account, you won't be able to access it again.