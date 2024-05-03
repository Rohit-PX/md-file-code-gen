package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/Rohit-PX/md-file-code-gen/markdownutils"

	"github.com/golang-commonmark/markdown"
)

var ArtifactDirectory string = "artifacts"
var YamlFileName string = ArtifactDirectory + "/docTest.yaml"
var KubectlCmdFileName string = ArtifactDirectory + "/kubectlCmd.sh"
var kubectlCmd string = "kubectl"
var pxctlCmd string = "pxctl"
var cmdType, ipAddr string

func main() {

	//Flag management
	var mdFilePath, kubeconfigFilePath string
	flag.StringVar(&mdFilePath, "mdfile", "", `Path to the md file`)
	flag.StringVar(&kubeconfigFilePath, "kubeconfig", "", `Path to the kubeconfig file`)
	flag.StringVar(&cmdType, "commandType", "", `Command type could be one of - kubectl or pxctl (pxctl needs to be run on the PX node directly)`)
	flag.StringVar(&ipAddr, "ipaddr", "", `IP address of worker node for pxctl commands`)
	flag.Parse()
	if mdFilePath == "" {
		log.Fatalln("Please, provide path to an mdfile for the readme to parse.")
	}

	body, err := ioutil.ReadFile(mdFilePath)
	if err != nil {
		log.Fatalf("unable to read file: %v", err)
	}
	md := markdown.New(markdown.XHTMLOutput(true), markdown.Nofollow(true))
	tokens := md.Parse(body)

	yamlFile, DocCmdFile, err := markdownutils.CreateArtifactFiles(ArtifactDirectory, YamlFileName, KubectlCmdFileName)
	if err != nil {
		log.Fatalf("unable to create yaml/cmd files: %v", err)
	}

	defer func() {
		yamlFile.Close()
		DocCmdFile.Close()
	}()
	//Print the result
	for _, t := range tokens {
		snippet := markdownutils.GetSnippet(t)

		if snippet.Content != "" {
			switch snippet.Lang {
			case "yaml":
				yamlFile.Write([]byte(fmt.Sprintf("---\n%s", snippet.Content)))
			case "bash":
				DocCmdFile.Write([]byte(fmt.Sprintf("\n%s", snippet.Content)))
			default:
				fmt.Println("Non executable snippet.")

			}
		}
	}

	// Read kubeconfig and set KUBECONFIG environment variable accordingly
	os.Setenv("KUBECONFIG", kubeconfigFilePath)

	// Apply yaml file and execute kubectl command script

	var cmd *exec.Cmd
	var out, kubeErr bytes.Buffer
	if cmdType == kubectlCmd {
		cmd = exec.Command("kubectl", "apply", "-f", YamlFileName)
		cmd.Stdout = &out
		cmd.Stderr = &kubeErr
	} else if cmdType == pxctlCmd {
		// open the file containing just pxctl commands
		file, err := os.Open(KubectlCmdFileName)
		if err != nil {
			log.Fatalf("Error opening file:", err)
		}
		// Create a new Scanner to read the file.
		scanner := bufio.NewScanner(file)
		// Set the split function to the default ScanLines
		scanner.Split(bufio.ScanLines)

		// Loop over all lines in the file.
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line)
			//pxSSH := fmt.Sprintf("ssh root@%s '%s'\n", ipAddr, line) // Print each line, or process it as needed.
			//cmd = exec.Command(pxSSH)
			//cmd.Stdout = &out
			//cmd.Stderr = &kubeErr

			//if err := cmd.Run(); err != nil {
			//	fmt.Println(out.String())
			//	log.Fatal(cmd.Stderr)
			//}
		}

		//cmd = exec.Command("bash", KubectlCmdFileName)
		//cmd.Stdout = &out
		//cmd.Stderr = &kubeErr

		//if err := cmd.Run(); err != nil {
		//	fmt.Println(out.String())
		//	log.Fatal(cmd.Stderr)
		//}
	}

	//if err := cmd.Run(); err != nil {
	//	log.Fatal(cmd.Stderr)
	//}
	//fmt.Println(out.String())

	//cmd = exec.Command("bash", KubectlCmdFileName)
	//cmd.Stdout = &out
	//cmd.Stderr = &kubeErr

	//if err := cmd.Run(); err != nil {
	//	log.Fatal(cmd.Stderr)
	//}
	//fmt.Println(out.String())
}
