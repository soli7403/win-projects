# DariTerm User Guide

DariTerm is a terminal app for people who work on servers. Use SSH, Telnet, serial devices and your PC's own shells (cmd, PowerShell, WSL) side by side in tabs and split panes. Server passwords and keys are encrypted and stored only on your PC.

> This guide covers version 1.0.0. 한국어: [user-guide.ko.md](./user-guide.ko.md)

---

## 1. Install

1. Run `DariTerm-amd64-installer.exe`.
   - If you see "Windows protected your PC", click **More info → Run anyway** (the installer is not code-signed yet).
2. Allow administrator access, accept the license, and click **Install**.
   - Install folder: `C:\Program Files\DariTerm`
   - A **DariTerm** shortcut is added to the Start menu and the desktop.
3. To uninstall: **Settings > Apps > Installed apps > DariTerm > Uninstall**. Your server list and settings are kept, so reinstalling picks up where you left off.

## 2. First launch

The first launch takes four short steps:

1. **Choose your language** — 10 languages. Your Windows language is preselected.
2. **Choose how you will use it** — *Personal* (free, no time limit) or *Business* (free 30-day evaluation, then a license). You can enter a business license key here if you have one.
3. **Create an app lock password** — at least 8 characters. It encrypts your server passwords and keys and is never sent anywhere.
4. **Save your recovery code** — a 24-character code shown **only once**. Print it or write it down. It is the only way to recover if you forget the password.

> 💡 If you used PuTTY or SecureCRT, DariTerm offers to import your saved sessions (see section 7).

## 3. A quick tour

```
┌───────────┬───────────────────────────────────────────┐
│ Icon row  │ Tab bar  [web-01] [db-01] [+]              │
├───────────┼───────────────────────────────────────────┤
│ Server    │                                           │
│ list      │        Terminal panes (can be split)       │
│ (sidebar) │                                           │
│ Snippets  │                                           │
├───────────┴───────────────────────────────────────────┤
│ Status bar: connection · usage · Unlocked              │
└───────────────────────────────────────────────────────┘
```

- **Top-left icons**: show/hide the server list, snippets, broadcast input to all panes, settings.
- **Top-right icon**: SFTP (file) panel.
- Click **"Unlocked"** at the bottom right to lock the app immediately.

## 4. Add a server and connect

### Add a server
1. Click **Add server** in the sidebar.
2. **General** tab: protocol (SSH/Telnet/Serial/Local), display name, address, port, user name.
   - Typing `user@10.0.0.11:22` in the address field fills in user, host and port for you.
   - Set **Environment** to *Production* to get a red border and an extra confirmation before dangerous commands.
   - A server with no color or environment of its own **inherits them from its folder**. If a folder is *Production*, the servers in it are protected as Production too (an environment set on the server itself wins).
3. **Sign-in** tab: password / key file / SSH agent / ask every time.
4. Click **Save and connect**.

### Ways to connect
- **Double-click** a server in the sidebar.
- Press **Ctrl+K** (Quick connect), type part of a name or IP, and press Enter.
  - Type `user@host` to connect to a server you haven't saved.
  - Type `>` for commands (new local terminal, open settings, switch theme, lock app, …).
- **Ctrl+Alt+T** opens a terminal on your PC.

### First connection to a server
You'll see "First connection to this server" with a key fingerprint. If it matches what your server administrator gave you, click **Trust and connect**. If the key changes later you'll get a warning — don't connect unless you know why.

### Save the password
When asked for a password, tick **"Save encrypted and sign in automatically next time"**.

## 5. Using the terminal

| To… | Do this |
|---|---|
| Copy | Select with the mouse — it's copied right away (or Ctrl+Shift+C) |
| Paste | Ctrl+V or right-click |
| **Send a screenshot to the server** | Copy a screenshot, then press **Ctrl+V** in the terminal. DariTerm uploads it and types its path — handy for AI CLIs such as Claude Code |
| Insert a file path | Drag a file from Explorer onto the terminal |
| Find text | Ctrl+Shift+F |
| Change font size | Ctrl+= bigger, Ctrl+- smaller, Ctrl+0 reset |
| New line without running | Shift+Enter |

> ⚠️ **Production protection**: on a *Production* server, commands like `rm -rf`, `reboot` or `DROP TABLE` are not run right away — DariTerm asks "Run it anyway?" first.

## 6. Tabs and split panes

| To… | Do this |
|---|---|
| Open the same server beside it | Ctrl+Shift+D (right), Ctrl+Shift+E (below) |
| Move between panes | Alt+Arrow keys |
| Maximize one pane | Ctrl+Shift+Enter |
| Switch tabs | Ctrl+Tab, Ctrl+1–9 |
| Type the same command into several servers | Ctrl+Shift+B (broadcast input on/off) |
| Close | Ctrl+Shift+W |
| Open every server in a folder | Right-click the folder → "Open all folder servers tiled" |
| Two side-by-side views | Right-click a tab → "Move to right side", or drag a tab to the right edge and drop it when "Drop here to split left and right" appears |

The tab you are looking at has a color bar on top (its server color, or blue). Other tabs get a light tint of their server color.

## 7. Import from other apps

Click **Import** in the sidebar. DariTerm looks for:

- PuTTY saved sessions
- SecureCRT session folders (including folder structure and port forwarding) — use **Choose SecureCRT folder** for another location
- `~/.ssh/config`
- AWS EC2 — **Load AWS EC2** (requires the aws CLI)

Tick what you want and click **Import N**. Passwords are not imported; enter each once on first connection and save it.

## 8. Transfer files (SFTP)

1. Connect over SSH, then press **Ctrl+Shift+S** (or the top-right icon).
2. The server's files appear on the right:
   - **Upload**: drag files from Explorer onto the list
   - **Download / edit**: right-click a file → Download / "Edit with an app on this PC" (saving uploads it automatically)
   - **Preview**: double-click a file
   - **Sort**: click Name, Size, or Modified above the list; click again to reverse (folders stay on top)
   - **Move terminal to this folder (cd)**
3. **Server-to-server copy**: drag a file from one server's tab to another server's tab. Progress appears in the copy list at the bottom of the window.
4. **If a transfer drops or is cancelled**: an existing file with the same name on the receiving side keeps its old content (DariTerm writes to a temporary `name.dtpart-xxxxxxxx` and renames it only when done). If a network drop leaves a `.dtpart-` file behind, delete it by hand.

## 9. Snippets

Save frequently used commands and run them with one click.

1. In the snippets area below the sidebar, click **New command**.
2. Enter a name and command (multiple lines allowed). Turn on "Run immediately after typing" to run it right away.
3. Click a snippet to type it into the current pane. In a tab with broadcast input on, it goes to every pane.

## 10. Staying secure

- **Auto-lock**: after 15 minutes without input, or when Windows locks, DariTerm locks itself. Connections stay open. Change the time in Settings > Security.
- **Unlock with Windows sign-in**: turn it on in Settings > Security.
- **Forgot the password?** On the lock screen, choose "Reset with recovery code".
- **Manage saved accounts**: Settings > Security > Saved accounts.

## 11. Using several PCs

- **Backup**: Settings > Backup → **Export backup**. On another PC, **Import backup** and unlock with the original PC's app lock password.
- **git sync**: enter a **private** git repository in Settings > git sync. DariTerm uploads one file encrypted with your app lock password to keep server lists in sync.

## 12. Settings at a glance

Settings (gear icon) are saved as soon as you change them.

- **Appearance**: language, theme (dark/light), terminal colors, font, cursor, maximum vertical panes
- **Input**: copy/paste behavior, confirm multi-line paste, production command warning
- **File transfer**: server upload folder for images, auto-delete old uploads
- **Connection**: reconnect when disconnected, restore last layout, log terminal output
- **git sync / Security / Backup / About**

## 13. Personal and business use

- **Personal use** (learning, personal projects, hobbies) is free with no time limit.
- **Business use** (work for a company, institution or organization, or commercial work) can be evaluated free for 30 days; after that each user needs a business license.
- Paste your license key in **Settings > About > Usage** and click **Apply**. The company name and number of licensed users are shown.
- After the evaluation ends the program keeps working; a notice appears once at startup.
- Licensing inquiries: **soli97403@gmail.com**

## 14. FAQ

**Q. My connection keeps dropping.**
With Settings > Connection "Reconnect when disconnected" on, DariTerm retries at 2, 4, 8, 16 and 30 seconds. After it stops, press Enter in the terminal to reconnect.

**Q. I want my work on the server to keep running after I close the window.**
Turn on **"Keep session (tmux)"** in the server editor > Advanced. The server needs tmux. Reopening brings you back to the same screen.

**Q. I need to go through a jump server.**
Server editor > Advanced > "Jump server (ProxyJump)": pick a saved server or enter one manually.

**Q. Port forwarding?**
Server editor > Port forwarding tab: add L (a port on your PC → an address reachable from the server) or R.

**Q. My PPK key won't load.**
Convert PPK v3 keys with PuTTYgen → Conversions → Export OpenSSH key.

**Q. Is any of my data sent anywhere?**
No. Server lists, passwords, keys and license checks are all handled on your PC.

---

## Keyboard shortcuts

These are the **default keys**. In Settings > Shortcuts you can change or clear the key of each action, and turning "Use shortcuts" off disables all app shortcuts so those keys go to the terminal (server) — useful when vi or tmux use the same keys. If you pick keys another action was using, that action is left without a key. The paste and copy keys cannot be changed and keep working when shortcuts are off.

| Keys | Action |
|---|---|
| Ctrl+K | Quick connect |
| Ctrl+Alt+T | New local terminal |
| Ctrl+Shift+W | Close pane |
| Ctrl+Shift+D / E | Same server to the right / below |
| Alt+Arrow keys | Move between panes |
| Ctrl+Shift+Enter | Maximize pane |
| Ctrl+Tab / Ctrl+1–9 | Switch tabs |
| Ctrl+V | Paste (auto-upload images and files) |
| Ctrl+Alt+V | Clipboard image → server |
| Ctrl+Shift+C | Copy |
| Ctrl+Shift+F | Find in terminal |
| Ctrl+Shift+S | SFTP panel |
| Ctrl+Shift+B | Broadcast input |
| Ctrl+Shift+L | Show/hide server list |
| Ctrl+= / - / 0 | Font size |
| F11 | Maximize/restore window |

© 2026 YangHun Lee (kai). All rights reserved.
