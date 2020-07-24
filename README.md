## Introduction

Stream exposes a resource-friendly WebDAV server to be used with Kodi and Infuse.
As a file source, a Google Drive Shared Drive must be given.

## Early Access

Stream currently "works", but offers a sub-optimal user experience.
Errors will get more friendly as the project grows and the CLI will get more useful as well.

To give early feedback, either [open an issue](https://github.com/m-rots/stream/issues/new) or [drop a message](https://discord.gg/tYqEWQ7) in my Discord server!

### Building the CLI

1. Install [Golang](https://golang.org/dl/) (version 1.13 or above).
2. Clone this repository and `cd` into it.
3. Run: `go build -o stream cmd/stream/main.go`

You should now see a binary called `stream` in the current working directory.

### Using the CLI

Make sure you create a Service Account which has read access to the Shared Drive in question.
Additionally, please check whether you have the Drive API enabled in Google Cloud.
Save a JSON key of this service account and store it next to the `stream` binary.

Next, identify the IDs of your `Movies` and `TV` folders in the Google Drive WebUI.

1. Head to your Shared Drive.

2. Open the `Movies` folder so you can see its contents.

3. Copy the last part of the URL and copy it somewhere.

    ```
    https://drive.google.com/drive/u/0/folders/<here is the ID>
    ```

4. Repeat steps 2 and 3 for your `TV` folder.

Now, create a new file next to the `stream` binary called `config.yml` and copy the following:

```yaml
# replace `account.json` with the name of your service account JSON file.
auth: account.json

# path does not matter, just keep it consistent
database: bernard.db

# See note below
depth: 1

# Replace Drive with your own Drive ID (same technique as for the folders)
drive: XXXXXXXXXXXXXXXXXVA

# Replace Films with the ID of your Movies folder
films: XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX

# Replace Shows with the ID of your TV folder
shows: XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
```

The `depth` value should be set at `1` if you do not have any folders in between the `TV` folder and the TV Show folders themselves.

Example: `/Media/TV/The Boys (2019)`

The `depth` value should be set at `2` if you do have an additional folder in between the `TV` folder and the TV Show folders themselves.

Example: `/Media/TV/Action/The Boys (2019)`

If you have multiple additional folders, adjust accordingly.

Congrats! That's all there is too it!
You can now start the server by running `./stream` from your terminal.

*Note: Stream will try to use port 3000 to boot the server. If you want to connect from outside your PC, either remember the IP address of your machine or use a reverse proxy such as [Caddy](https://caddyserver.com/v2).*

### Connecting with Kodi

1. Head over to Settings (the cogwheel)
2. Click on the media tab
3. Within manage sources, hit the “videos…” button
4. Click the “browse” button within the popup window
5. Make sure to scroll down to find the “add network location…” button
6. Select the WebDAV protocol and enter in `http://localhost:3000` for the server address.

### Connecting with Infuse

I recommend you use [Caddy](https://caddyserver.com/v2) as a reverse proxy for connecting with Infuse.

1. Head over to Settings
2. Click the ‘Add Files’ button
3. Select ‘From a shared Folder’
4. Input the credentials listed up above.
5. Select WebDAV protocol and enter in the server address.

## FAQ

> Does Stream support multiple Shared Drives?

Not at the moment, it might in the future.

> Does Stream filter out any files other than MP4 files and MKV files?

Not at the moment, it might in the future.

> What port does Stream open?

Port 3000. This might be changed in the future.