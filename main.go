package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	accessToken := flag.String("accessToken", "", "Matrix access token")
	outFile := flag.String("elementDesktopJs", "", "Output file for Element Desktop's JS Console")
	flag.Parse()

	syncv3server := os.Getenv("SYNCV3_SERVER")
	syncv3db := os.Getenv("SYNCV3_DB")

	exit1 := false
	if *accessToken == "" {
		fmt.Println("E: Missing -accessToken command line flag")
		exit1 = true
	}
	if *outFile == "" {
		fmt.Println("E: Missing -elementDesktopJs command line flag")
		exit1 = true
	}
	if syncv3server == "" {
		fmt.Println("E: Missing SYNCV3_SERVER environment variable")
		exit1 = true
	}
	if syncv3db == "" {
		fmt.Println("E: Missing SYNCV3_DB environment variable")
		exit1 = true
	}

	if exit1 {
		os.Exit(1)
	}
	// end params validation

	fmt.Println("Homeserver URL:", syncv3server)

	fmt.Println("Identifying user and device IDs...")
	userId, deviceId, err := getUserInfo(syncv3server, *accessToken)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	fmt.Println("User ID:", userId)
	fmt.Println("Device ID:", deviceId)

	if userId == "" || deviceId == "" {
		fmt.Println("E: User or Device ID is empty")
		os.Exit(2)
	}

	fmt.Println("Cleaning up sync loop in proxy...")
	err = deleteSyncV3Device(syncv3db, userId, deviceId)
	if err != nil {
		fmt.Println(err)
		os.Exit(3)
	}

	fmt.Println("Extracting to-device messages...")
	messages, err := getSyncV3DeviceMessages(syncv3db, userId, deviceId)
	if err != nil {
		fmt.Println(err)
		os.Exit(3)
	}
	fmt.Printf("Got %d messages to re-send.", len(messages))
	fmt.Println()

	desktopSyncJs := "// Copy and paste this whole file into your Element Desktop JS Console\n"
	for i, msg := range messages {
		fmt.Printf("Copying %s message from %s (%d/%d)...", msg.EventType, msg.Sender, i, len(messages))

		desktopSyncJs += fmt.Sprintf("console.log('Importing %s message from %s (%d/%d)');\n", msg.EventType, msg.Sender, i, len(messages))
		desktopSyncJs += "await mxMatrixClientPeg.get().syncApi.processSyncResponse({}, {\"to_device\": {\"events\": ["
		desktopSyncJs += msg.Message
		desktopSyncJs += "]}});\n"

		fmt.Println("OK")
	}

	fmt.Println("Writing JS file for Element Desktop...")
	f, err := os.Create(*outFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(5)
	}
	defer f.Close()
	_, err = f.Write([]byte(desktopSyncJs))
	if err != nil {
		fmt.Println(err)
		os.Exit(5)
	}

	fmt.Println("Done! You can start your sliding sync proxy now. You will need to manually copy/paste the Element Desktop JS file.")
}

func getUserInfo(csApi string, accessToken string) (string, string, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/_matrix/client/v3/account/whoami", csApi), nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	j := make(map[string]interface{})
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&j)
	if err != nil {
		return "", "", err
	}
	return j["user_id"].(string), j["device_id"].(string), nil
}

func deleteSyncV3Device(syncv3db string, userId string, deviceId string) error {
	db, err := sql.Open("postgres", syncv3db)
	if err != nil {
		return err
	}
	defer db.Close()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	_, err = tx.Exec("DELETE FROM syncv3_sync2_devices WHERE user_id = $1 AND device_id = $2", userId, deviceId)
	if err != nil {
		return err
	}
	_, err = tx.Exec("DELETE FROM syncv3_sync2_tokens WHERE user_id = $1 AND device_id = $2", userId, deviceId)
	if err != nil {
		return err
	}
	return tx.Commit()
}

type deviceMessage struct {
	EventType string
	Sender    string
	Message   string
}

func getSyncV3DeviceMessages(syncv3db string, userId string, deviceId string) ([]*deviceMessage, error) {
	db, err := sql.Open("postgres", syncv3db)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query("SELECT event_type, sender, message FROM syncv3_to_device_messages WHERE user_id = $1 AND device_id = $2 ORDER BY position ASC", userId, deviceId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	messages := make([]*deviceMessage, 0)
	for rows.Next() {
		msg := &deviceMessage{}
		err = rows.Scan(&msg.EventType, &msg.Sender, &msg.Message)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}
