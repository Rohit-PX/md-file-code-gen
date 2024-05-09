package markdownutils

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/golang-commonmark/markdown"
	"golang.org/x/crypto/ssh"
)

const (
	kubectlCmd        = "kubectl"
	pxctlCmd          = "pxctl"
	ArtifactDirectory = "artifacts"
	YamlFileName      = ArtifactDirectory + "/docTest.yaml"
	CmdFileName       = ArtifactDirectory + "/commands.sh"
	NodeUser          = "root"
	NodePassword      = "Password1"
)

// Snippet represents the snippet we will output.
type Snippet struct {
	Content string
	Lang    string
}

// ExecutableInfo contains information about commands and where to execute them
type ExecutableInfo struct {
	CommandType string `json:"commandtype"`
	IpAddr      string `json:"ipaddr"`
}

type Conn struct {
	Session ssh.Session
	ExecutableInfo
}

// getSnippet extract only code Snippet from markdown object.
func GetSnippet(tok markdown.Token) Snippet {
	switch tok := tok.(type) {
	case *markdown.CodeBlock:
		return Snippet{
			tok.Content,
			"code",
		}
	case *markdown.CodeInline:
		return Snippet{
			tok.Content,
			"code inline",
		}
	case *markdown.Fence:
		return Snippet{
			tok.Content,
			tok.Params,
		}
	}
	return Snippet{}
}

// CreateArtifactFiles creates directories, yaml and sh files required for testing the doc
func CreateArtifactFiles(dirName, yamlFileName, cmdFileName string) (*os.File, *os.File, error) {
	err := os.MkdirAll("artifacts", os.ModePerm)
	if err != nil {
		return nil, nil, err
	}

	yamlFile, err := os.Create(yamlFileName)
	if err != nil {
		return nil, nil, err
	}

	cmdFile, err := os.Create(cmdFileName)
	if err != nil {
		return nil, nil, err
	}

	return yamlFile, cmdFile, err
}

// ExecuteCmdFile
//
//		for kubectl: 1. applies yaml file and
//	              2. executes kubectl commands from command file using provided kubeconfig file
//		for pxctl: executes pxctl commands by creating ssh connection to the provided ip addres
func ExecuteCmdFile(execInfo *ExecutableInfo) error {
	var cmd *exec.Cmd
	var out, kubeErr bytes.Buffer
	myConn := NewConnection(execInfo.IpAddr)
	switch execInfo.CommandType {
	case kubectlCmd:
		log.Printf("Kubectl command type selected.")
		cmd = exec.Command("kubectl", "apply", "-f", YamlFileName)
		cmd.Stdout = &out
		cmd.Stderr = &kubeErr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to execute kubectl command error: %v", cmd.Stderr)
		}
		fmt.Println(out.String())

		err := ReadFileRunCmd(myConn, CmdFileName, false)
		if err != nil {
			return err
		}

	case pxctlCmd:
		log.Printf("pxctl command type selected. Reading file: %s", CmdFileName)
		file, err := os.Open(CmdFileName)
		if err != nil {
			log.Fatalf("Error opening file: %v", err)
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				log.Printf("pxctl command: %s", line)
				out, err := myConn.RunSSHCmd(line)
				if err != nil {
					return fmt.Errorf("failed to run command: \"%s\" on: %s, err: %v", line, execInfo.IpAddr, err)
				}
				log.Printf("PXCTL output: %s", out)
			}
		}
	}
	return nil
}

func ReadFile(fileName string) {
	data, err := os.ReadFile(fileName)
	if err != nil {
		log.Panicf("failed reading data from file: %s", err)
	}
	fmt.Println(data)
}

func NewConnection(ip string) Conn {
	sess, err := GetSSHSession(ip)
	if err != nil {
		log.Fatalf("failed to get new session for %s: err: %v", ip, err)
	}
	return Conn{Session: sess}
}

func GetSSHSession(ip string) (ssh.Session, error) {
	config := &ssh.ClientConfig{
		User: NodeUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(NodePassword),
		},

		// It is very important to verify the server's host key in production
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Connect to the SSH server
	connection, err := ssh.Dial("tcp", ip+":22", config)
	if err != nil {
		return ssh.Session{}, fmt.Errorf("failed to dial: %s. err: %v", ip, err)
	}

	// Open a session
	newSession, err := connection.NewSession()
	if err != nil {
		return ssh.Session{}, fmt.Errorf("failed to create session: %s. err: %v", ip, err)
	}

	return *newSession, nil
}

func (c *Conn) RunSSHCmd(cmd string) (string, error) {
	output, err := c.Session.CombinedOutput(cmd)
	return string(output), err
}

func RunCmd(cmd string) (string, error) {
	var execCmd *exec.Cmd
	var out, kubeErr bytes.Buffer

	execCmd = exec.Command(cmd)
	execCmd.Stdout = &out
	execCmd.Stderr = &kubeErr

	if err := execCmd.Run(); err != nil {
		return out.String(), fmt.Errorf("failed to execute cmd: %s, err: %v", cmd, err)
	}
	return out.String(), nil
}

func ReadFileRunCmd(myConn Conn, CmdFileName string, isSSH bool) error {
	file, err := os.Open(CmdFileName)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			if isSSH {
				out, err := myConn.RunSSHCmd(line)
				if err != nil {
					return fmt.Errorf("failed to run command: \"%s\" on: %s, err: %v", line, myConn.IpAddr, err)
				}
				log.Printf("PXCTL output: %s", out)
			} else {
				out, err := RunCmd(line)
				if err != nil {
					return fmt.Errorf("failed to run command: \"%s\" on: %s, err: %v", line, myConn.IpAddr, err)
				}
				log.Printf("PXCTL output: %s", out)

			}
		}
	}
	return nil
}

//// readFromWeb call the given url and return the content of the readme.
//func readFromWeb(url string) ([]byte, error) {
//	resp, err := http.Get(url)
//	if err != nil {
//		return nil, err
//	}
//	defer resp.Body.Close()
//
//	return ioutil.ReadAll(resp.Body)
//}
