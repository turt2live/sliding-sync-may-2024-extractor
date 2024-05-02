# sliding-sync-may-2024-extractor
Extract to-device messages out the sliding sync proxy

## Usage

> [!WARNING]
> This has not been widely tested. Use at your own risk.

1. Stop your sliding sync proxy
2. Pick and download the executable for your platform from `./bin` (or compile it yourself with `go build`)
3. Run `SYNCV3_SERVER="..." SYNCV3_DB="..." ./sliding-sync-may-2024-extractor -accessToken "..." -elementDesktopJs ./run_in_console.js`
   using the same environment variables from your proxy, and `accessToken` for the device you wish to fix. `SYNCV3_SECRET`
   is not required.
4. Start your sliding sync proxy again
