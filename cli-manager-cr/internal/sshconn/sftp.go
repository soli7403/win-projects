package sshconn

import (
	"github.com/pkg/sftp"
)

// SftpClient 는 이 연결의 sftp 클라이언트를 돌려준다. 처음 부를 때 sftp 하위 시스템을 연다 (protocols.md §3).
// 연결이 끊기면 새 연결의 클라이언트가 따로 만들어진다.
func (pConn *Conn) SftpClient() (*sftp.Client, error) {
	pConn.mtxSftp.Lock()
	defer pConn.mtxSftp.Unlock()
	if pConn.sftpClient != nil {
		return pConn.sftpClient, nil
	}
	session, errSession := pConn.Client.NewSession()
	if errSession != nil {
		return nil, errSession
	}
	stdin, errIn := session.StdinPipe()
	if errIn != nil {
		_ = session.Close()
		return nil, errIn
	}
	stdout, errOut := session.StdoutPipe()
	if errOut != nil {
		_ = session.Close()
		return nil, errOut
	}
	if errSub := session.RequestSubsystem("sftp"); errSub != nil {
		_ = session.Close()
		return nil, errSub
	}
	client, errClient := sftp.NewClientPipe(stdout, stdin)
	if errClient != nil {
		_ = session.Close()
		return nil, errClient
	}
	pConn.sftpClient = client
	return client, nil
}
