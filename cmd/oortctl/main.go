package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/darlingson/Oort-Object-Storage/internal/client"
)

var (
	serverURL string
	apiKey    string
)

var rootCmd = &cobra.Command{
	Use:   "oortctl",
	Short: "Oort Object Storage CLI",
}

func makeClient() *client.Client {
	c := client.New(serverURL)

	if apiKey != "" {
		c.APIKey = apiKey
		return c
	}

	token, err := client.ReadToken()
	if err == nil {
		c.JWT = token
	}

	return c
}

func execute() {
	rootCmd.PersistentFlags().StringVarP(
		&serverURL, "server", "s", "http://localhost:3333", "server URL",
	)
	rootCmd.PersistentFlags().StringVar(
		&apiKey, "api-key", "", "API key for machine auth",
	)

	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(bucketCmd)
	rootCmd.AddCommand(uploadCmd)
	rootCmd.AddCommand(downloadCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(signCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	execute()
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login and store JWT token",
	RunE: func(cmd *cobra.Command, args []string) error {
		email, _ := cmd.Flags().GetString("email")
		password, _ := cmd.Flags().GetString("password")

		if email == "" || password == "" {
			fmt.Print("Email: ")
			fmt.Scanln(&email)
			fmt.Print("Password: ")
			fmt.Scanln(&password)
		}

		c := client.New(serverURL)
		result, err := c.Login(email, password)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		if err := client.WriteToken(result.Token); err != nil {
			return fmt.Errorf("failed to store token: %w", err)
		}

		fmt.Printf("Logged in as %s\n", result.User.Email)
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored JWT token",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := client.DeleteToken(); err != nil {
			return fmt.Errorf("logout failed: %w", err)
		}
		fmt.Println("Logged out")
		return nil
	},
}

var bucketCmd = &cobra.Command{
	Use:   "bucket",
	Short: "Manage buckets",
}

var bucketCreateCmd = &cobra.Command{
	Use:   "create",
	Args:  cobra.ExactArgs(1),
	Short: "Create a bucket",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := makeClient()
		bucket, err := c.CreateBucket(args[0])
		if err != nil {
			return err
		}
		fmt.Printf("Created bucket %s (%s)\n", bucket.Name, bucket.ID)
		return nil
	},
}

var bucketListCmd = &cobra.Command{
	Use:   "list",
	Short: "List buckets",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := makeClient()
		buckets, err := c.ListBuckets()
		if err != nil {
			return err
		}
		for _, b := range buckets {
			fmt.Printf("%s  %s\n", b.Name, b.ID)
		}
		return nil
	},
}

var uploadCmd = &cobra.Command{
	Use:   "upload <bucket> <file>",
	Args:  cobra.ExactArgs(2),
	Short: "Upload a file to a bucket",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := makeClient()
		bucket, filePath := args[0], args[1]

		f, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer f.Close()

		obj, err := c.UploadObject(bucket, filePath, f)
		if err != nil {
			return err
		}

		fmt.Printf("Uploaded %s (%d bytes)\n", obj.ObjectKey, obj.SizeBytes)
		return nil
	},
}

var downloadCmd = &cobra.Command{
	Use:   "download <bucket> <key>",
	Args:  cobra.ExactArgs(2),
	Short: "Download an object",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := makeClient()
		bucket, key := args[0], args[1]

		output, _ := cmd.Flags().GetString("output")

		resp, err := c.DownloadObject(bucket, key)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(body)))
		}

		if output == "" {
			output = filepath.Base(key)
		}

		out, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer out.Close()

		written, err := io.Copy(out, resp.Body)
		if err != nil {
			return fmt.Errorf("download failed: %w", err)
		}

		fmt.Printf("Downloaded %s (%d bytes)\n", output, written)
		return nil
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete <bucket> <key>",
	Args:  cobra.ExactArgs(2),
	Short: "Delete an object",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := makeClient()
		err := c.DeleteObject(args[0], args[1])
		if err != nil {
			return err
		}
		fmt.Println("Deleted")
		return nil
	},
}

var lsCmd = &cobra.Command{
	Use:   "ls <bucket>",
	Args:  cobra.ExactArgs(1),
	Short: "List objects in a bucket",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := makeClient()
		objects, err := c.ListObjects(args[0])
		if err != nil {
			return err
		}
		for _, o := range objects {
			fmt.Printf("%s  (%d bytes, %s)\n", o.ObjectKey, o.SizeBytes, o.ContentType)
		}
		return nil
	},
}

var signCmd = &cobra.Command{
	Use:   "sign <bucket> <key>",
	Args:  cobra.ExactArgs(2),
	Short: "Generate a signed URL",
	RunE: func(cmd *cobra.Command, args []string) error {
		operation, _ := cmd.Flags().GetString("op")
		expiresIn, _ := cmd.Flags().GetString("expires")

		if operation != "upload" && operation != "download" {
			return fmt.Errorf("operation must be 'upload' or 'download'")
		}

		c := makeClient()
		result, err := c.SignURL(args[0], args[1], operation, expiresIn)
		if err != nil {
			return err
		}

		fmt.Println(result.URL)
		return nil
	},
}

func init() {
	loginCmd.Flags().StringP("email", "e", "", "email address")
	loginCmd.Flags().StringP("password", "p", "", "password")

	bucketCmd.AddCommand(bucketCreateCmd)
	bucketCmd.AddCommand(bucketListCmd)

	downloadCmd.Flags().StringP("output", "o", "", "output file path")

	signCmd.Flags().StringP("op", "m", "", "operation: upload or download (required)")
	signCmd.Flags().StringP("expires", "e", "1h", "expiry duration (e.g. 24h)")
	signCmd.MarkFlagRequired("op")
}
