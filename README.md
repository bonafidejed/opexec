# opexec
a 1Password binary for OpenClaw SecretRef

## Background
I couldn't find a good way to run a "secret provider" in OpenClaw for 1Password that I liked. All the ways I found required you to setup one provider for each secret due to the way the 1Password CLI works. So I made my own! It requires a 1Password service account and can connect directly or use a 1Password Connect Server.

### Requirements
* The necessary tools to build a Golang program installed. Check docuemntation specific for your distribution or look at https://go.dev/doc/install.
* Git
* Paid 1Password account
* Properly setup Service Account with Read access to the Vault in question.
* [Optional] a local "Connect Server" with an Access Token (can run in docker on the same host as OpenClaw)

## Download, Build, Install
### Assumptions
My OpenClaw is running on Linux in user space. I'm assuming yours is too, so check the following information for fitness to your setup before you use it.

### Download and Build
1. Clone the project into the folder of your choice.
1. At a shell prompt inside the folder issue this command: `go build -o ~/.local/bin/opexec`
1. At the same shell prompt, change the permissions so OpenClaw will like it: `chmod 755 ~/.local/bin/opexec`
1. At the same shell prompt, run this command and note the absolute path of your installation: `ls -l ~/.local/bin/opexec`. I'll refer to this as `<your-absolute-path>` but you have to substitute your value. It should look something like "/home/user/.local/bin/opexec".

### OpenClaw Environment Variable(s) for 1Password
To get opexec to connect to 1Password, you have to add information in an environment variable. If your OpenClaw gateway is running as a systemd gateway, you can put this in `~/.openclaw/gateway.systemd.env`. Otherwise, you can probably use `~/.openclaw/.env`.

If you're connecting to 1Password directly:
```
OP_SERVICE_ACCOUNT_TOKEN=<service-account-token>
```

If you're using a Connect Server you need two variables. If you're running it in docker on the same host your URL is probably `http://localhost:8080`.
```
OP_CONNECT_HOST=<connect-url-with-protocol-and-port>
OP_CONNECT_TOKEN=<api-key-token>
```

You probably need to restart your gateway after you make this change. On any shell prompt: `openclaw gateway restart`.

### OpenClaw Config
There are command line tools for configuration and secrets configuration, but I didn't master them. This is how to modify (add to) your openclaw.json. At a shell prompt in the openclaw folder, first make a backup: `cp openclaw.json openclaw-backup.json`.

Then add the following lines into the config and put in the value for `<your-absolute-path>` that you noted above. NOTE, if you're using a Connect Server, then you should change this line: `"passEnv": ["OP_CONNECT_HOST","OP_CONNECT_TOKEN"],`. If you want to use a Connect Server with 1Password as a backup, include all three.
```
... more JSON
  "secrets": {
    "providers": {
      "opexec": {
        "source": "exec",
        "command": "<your-absolute-path>",
        "timeoutMs": 10000,
        "passEnv": ["OP_SERVICE_ACCOUNT_TOKEN"],
        "jsonOnly": true
      }
    }
  }
... more JSON
```

Now you're ready to use the secrets provider. NOTE -- not all OpenClaw configuration values are eligable for a SecretRef, so see https://docs.openclaw.ai/reference/secretref-credential-surface for more information.

At the right place in your file, udpate a line to look like this. (You need to customize the key and the value to suit that particular part of your config.)
```
...more JSON...
"apiKey": { "source": "exec", "provider": "opexec", "id": "op://Vault/Item/field" }
...more JSON...
```

See below for information about the 1Password "reference" that goes into the "id" field.

## Appendix
### How opexec works
The "id" that you configure in the openclaw.json is a 1Password Reference to a particular vault, item and field. Again, it's up to you to properly authorize your Service Account or Connect Server Access Token to be able to read a vault. I actually recommend doing a Connect Server. It's pretty easy to setup and all you need to to to run it is download a single docker compose file, download a json credentials file, and spin-up the container. If you do it right you'll be able to connect to `http://localhost:8080` in minutes! See this page and scroll down to the bottom and click "Get Started"! https://www.1password.dev/connect

The program gets all of it's informaiton from the environment variables. Thes are the same variables used by the 1Password CLI. You cannot specify any of these options on the command line. If you provide `OP_SERVICE_ACCOUNT_TOKEN` it will connect directly to 1Password. If you provide both `OP_CONNECT_HOST` and `OP_CONNECT_TOKEN` it will connect to the Connect Server. If you provide all of them, it will try to connect to the Connect Server first and try to connect to 1Password if there is an error.

This program always prints information to the stderr. If you also want it to create a log file in the current working directory for debugging purposes, specify `-file` on the command line.

This program refers to secrets by their 1Password "reference." You can get the 1Password Reference from the desktop UI by right-clicking the field and choosing "Copy Secret Reference". The format of the 1Password Reference seems to be `op://<Vault>/<Item>/[Section>/]<FieldLabelOrID>`.

* NOTE that 1Password Secret References include the section name. I recommend copying the 1Password Secret Reference out of the 1Password UI instead of trying to construct it on your own.
* NOTE, it is possbile for collisions to occur when there are fields with the same label. When there is only one field with the label in that section, you'll get a secret reference that references the label, like op://Vault/Item/MyField. But if you add another field that's also named "MyField" in the same section, that reference won't work anymore and will cause an error. In such cases, both fields must be referred to by their ID instead of their label: eg, op://Vault/Item/ahwueiposjkl23hjkw7832jlre and op://Vault/Item/whjk378h23jklwejklk6jklrew.

### A Bit about the OpenClaw SecretRef
By default this takes JSON on the standard input, looks up the secret information, and provides a JSON response back. This is all defined by OpenClaw, see https://docs.openclaw.ai/gateway/secrets#secretref-contract. For your reference (not used by the project) relevent payloads are stored in this project in `payloads/example_request_payload.json` and `payloads/example_response_payload.json` repsectively.
  * As mentioned earlier, not all OpenClaw configuration values are eligable for a SecretRef, so see https://docs.openclaw.ai/reference/secretref-credential-surface for more information.

### Payloads for Development and Testing
First look in the `payloads` folder and make a copy of `example_request_payload.json` called `payload.json`. Then you have to go into that file and change the reference to something that's valid for your 1Password. Basically, change the 1Password "references" (described above) by changing what's in the "ids". Here's an example:
```
{ "protocolVersion": 1, "provider": "opexec", "ids": ["op://Bonafidejed/OpenAI/credential","op://Bonafidejed/Telegram/token","op://Bonafidejed/Slack/userToken"] }
```

### Visual Studio Code
I developed this on my mac with Visual Studio Code. Below is an example `.vscode/launch.json` to start from. Make sure to insert your 1Password service account token (or add the two environment variables for you Connect Server. Or provide all three.)
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

### Command Line Testing
To test from a command line, make sure you setup your environment variable(s) for that command line session. I import mine by creating an alias in my `.bashrc` that does this. How you approach that is up to you.
```
set -a; source ~/.openclaw/gateway.systemd.env; set +a
```

If you want to test right from the project folder, first do this any time you make changes. It will create the binary right there in the same folder.
```
go build
```

Then you can do a test like this from the command line:
```
cat payloads/payload.json | ./opexec
```
