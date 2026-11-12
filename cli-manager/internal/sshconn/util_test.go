package sshconn

import "fmt"

func fmtSscan(s string, p *int) (int, error) { return fmt.Sscan(s, p) }
