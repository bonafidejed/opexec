# opexec
a 1Password binary for OpenClaw SecretRef

## Background
There are lots of ways on the internet to use 1Password with OpenClaw. The documentation refers a lot to keeping your secrets safe. One reason I wanted to do it this way was that the alternative seemed to be having these secrets loaded into memory for the OpenClaw agents to see. If anything got compromised, this could be unsafe. There are other suggestions about how to integrate 1Password, none fit what I really wanted it to do. Most of them require creating one "secret provider" for each secret you want to use in your `openclaw.json` config and that just seemed awkward to me. The exec provider must be a standalone binary (which I didn't realize until I built one in python) so I decided to learn golang and make this.

### 1Password
This is intended for people that are already using 1Password. You can look at https://www.1password.dev/get-started$0 for a lot of information, but here's what I'd recommend to get started.

* Create a new Vault for the secrets for OpenClaw.
* Create a Service Account https://www.1password.dev/service-accounts/get-started$0
* Authorize the Service Account to have read access to the new Vault. (For this use, you only need read access to only that one Vault.)
* Save the API Key for the account, you won't be able to access it again.

After I had this running for a while, I wanted a faster response and I wanted OpenClaw to be able to start the gateway even if there was a problem with the 1Password API. I got their 1Password Connect Server going in Docker (on the same host as my OpenClaw) using the instructions here https://www.1password.dev/connect/get-started. This is toally optional and does work, but isn't the focus of this document.

To initiate connections, this program uses the same environment variable(s) as the 1Password CLI. They are not configurable in any other way. You may need to set them for your development environment, in any shell you're testing in, and for how your OpenClaw gateway runs.

* To use the regular API, configure OP_SERVICE_ACCOUNT_TOKEN which is in all the examples below.
* To user the Connect API, configure OP_CONNECT_HOST and OP_CONNECT_TOKEN.
* NOTE -- if you configure both, this program will try the Connect API first. If that has an error and the other environment variable is set, it will try the 1Password API. However, this can take longer and may timeout.

This program refers to secrets by their 1Password "reference." You can get the 1Password Reference from the desktop UI by right-clicking the field and choosing "Copy Secret Reference". The format of the 1Password Reference seems to be `op://<Vault>/<Item>/[Section>/]<FieldLabelOrID>`.

* NOTE that 1Password Secret References include the section name. I recommend copying the 1Password Secret Reference out of the 1Password UI instead of trying to construct it on your own.
* NOTE, it is possbile for collisions to occur when there are fields with the same label. When there is only one field with the label, you'll get a secret reference that references the label, like op://Vault/Item/MyField. But if you add another field that's also named "MyField" that won't work anymore and will cause an error. In that case, both fields must be referred to by their ID instead of their label: eg, op://Vault/Item/ahwueiposjkl23hjkw7832jlre and op://Vault/Item/whjk378h23jklwejklk6jklrew.

### OpenClaw
If you don't know what OpenClaw is, totally cool, please keep scrolling to the next github project, this one isn't for you.

By default this takes JSON on the standard input, looks up the secret information, and provides a JSON response back. This is all defined by OpenClaw, see https://docs.openclaw.ai/gateway/secrets#secretref-contract. For your reference (not used by the project) relevent payloads are stored in this project in `payloads/example_request_payload.json` and `payloads/example_response_payload.json` repsectively.
  * NOTE, not all OpenClaw configuration values are eligable for a SecretRef, so see https://docs.openclaw.ai/reference/secretref-credential-surface for more information.

## Buildling and Installing for OpenClaw
This is my first golang project, so I'm sure there are more elegant ways to set this up. If you know golang but are new to OpenClaw, it's important to remember that it runs in user space; it's not meant to be installed for all users. Hence, my ideas about how this can work are based on that.

Since everytihng's running in user space, the `bash` commands use filenames relative to `~` and the systemd unit filenames are relative to `%h`.

What I did on my installation (debian-based headless):

* clone the repo to a folder in my home directory
* go into the opexec folder
* run this command `go build -o ~/.local/bin/opexec`
* fix the permissions so OpenClaw is happy `chmod 755 ~/.local/bin/opexec`


## Running for OpenClaw
* When the `openclaw.json` configuration is updated, it must have the absolute path to the binary. I used `ls -l ~/.local/bin/opexec` to find the full path.
* I had errors if the permissions on the binary were not restricted enough. Specific directions are included below to address this.
* When connecting directly to 1Password, I had issues with the time it took to get the secrets back. (A) this is why I started using Connect, but also (B) I learned I had to set the timeout in the OpenClaw config. You'll see that below.

### Environment Variable(s)
On my production machine, I stored the variable(s) in a secure location on my machine. Open your favorite editor and put your variable(s) in `~/path/to/your/service-account.env`:

```
OP_SERVICE_ACCOUNT_TOKEN=<service-account-token>
```

* Be sure to set the permissions by doing `chmod 600 ~/path/to/your/service-account.env`
* I import this into my session by adding these lines to the bottom of my `.bashrc`
```
set -a
source ~/path/to/your/service-account.env
set +a
```
* I import this for my OpenClaw in the user service definition because my installation uses systemd. If your installation uses something different, you may need a different approach. The command I use is `systemctl --user edit openclaw-gateway.service` and then you have to add this one line:
```
EnvironmentFile=%h/path/to/your/service-account.env
```
* After editing the systemd service, you have to do this:
```
systemctl --user daemon-reload
systemctl --user restart openclaw-gateway.service
```

### Setting-Up the OpenClaw Config
I edit the `openclaw.json` file manually. It may be possible to do this with the included tools and if I find out the exact details I'll put that informaiton here. Until then here's how it can be done manually. You have to reference the absolute path of the executable, you should be able to get that by doing  and using that in the "command" below.

* NOTE if you're using the Connect API, you should change the "passEnv" below to be ["OP_CONNECT_HOST","OP_CONNECT_TOKEN"].
* Add (or update) a section in your `openclaw.json` file for the new secrets provider:
```
... more JSON
  "secrets": {
    "providers": {
      "opexec": {
        "source": "exec",
        "command": "/home/user/.local/bin/opexec",
        "timeoutMs": 10000,
        "passEnv": ["OP_SERVICE_ACCOUNT_TOKEN"],
        "jsonOnly": true
      }
    }
  }
... more JSON
```

* REMEMBER, not all OpenClaw configuration values are eligable for a SecretRef.
* Update an eligable entry in the `openclaw.json` to use your new provider. 

```
...more JSON...
"apiKey": { "source": "exec", "provider": "opexec", "id": "op://Vault/Item/field" }
...more JSON...
```

## Developing and Testing
By default this program prints out no information because that would interrupt the way the program is supposed to work. If you want to make it print to the stderr, add the "-screen" argument. If you want it to make an opexec.log file, add the "-file" agrument. You may not specify more than one argument. You may not specify a filename; it creates `opexec.log` file in the current working directory.

### Payloads
First look in the `payloads` folder and make a copy of `example_request_payload.json` called `payload.json`. Then you have to go into that file and change the reference to something that's valid for your 1Password. Basically, change the 1Password "reference" (described above) by changing what's in the "ids".

### Visual Studio Code
I developed this on my mac with Visual Studio Code. Below is an example `.vscode/launch.json` to start from. Make sure to insert your 1Password service account token.
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
            },
            "args": ["-screen"]
        }
    ]
}
```

### General Testing Information
To test from a command line, make sure you setup your environment variable(s) for that command line session and created the needed payload.

If you want to test right from the project folder, first do this any time you make changes:
```
go build
```

Then you can do a test like this from the command line:
```
cat payloads/payload.json | ./opexec
```
