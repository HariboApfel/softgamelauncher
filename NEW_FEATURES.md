# New Features - Delete Games & Command Line Launch

## 🗑️ Delete Games Feature

### **GUI Version**
- **Delete Button**: Added a delete button (🗑️) to the toolbar
- **Selection Required**: You must select a game in the list before deleting
- **Confirmation Dialog**: Shows a confirmation dialog to prevent accidental deletions
- **Success Feedback**: Displays a success message after deletion

### **Console Version**
- **Menu Option**: Added "6. Delete Game" to the main menu
- **Game Selection**: Lists all games and asks for the number to delete
- **Confirmation**: Requires "y" or "yes" confirmation before deletion
- **Success Feedback**: Shows confirmation message after successful deletion

## 🚀 Command Line Launch Options

### **CLI Version (`gamelauncher_cli.exe`)**
A separate command-line interface that doesn't require the GUI framework.

#### **Available Commands:**
- `-game <number>` - Launch a game by its number in the list
- `-list` - List all available games with their details
- `-help` - Show command-line usage information

#### **Examples:**
```bash
# Show help
gamelauncher_cli.exe -help

# List all games
gamelauncher_cli.exe -list

# Launch the first game
gamelauncher_cli.exe -game 1

# Launch the second game
gamelauncher_cli.exe -game 2
```

### **GUI Version with CLI Support (`gamelauncher.exe`)**
The main GUI application also supports command-line arguments when run from the command line.

#### **Usage:**
```bash
# Normal GUI mode
gamelauncher.exe

# Command-line mode (same options as CLI version)
gamelauncher.exe -game 1
gamelauncher.exe -list
gamelauncher.exe -help
```

## 📋 How to Use

### **Deleting Games**

#### **In GUI:**
1. Select a game from the list
2. Click the delete button (🗑️) in the toolbar
3. Confirm the deletion in the dialog
4. Game is removed and list is updated

#### **In Console:**
1. Choose option "6. Delete Game"
2. Select the game number from the list
3. Type "y" to confirm deletion
4. Game is removed from the list

### **Command Line Launch**

#### **Quick Launch:**
```bash
# Launch your first game instantly
gamelauncher_cli.exe -game 1
```

#### **Check Available Games:**
```bash
# See what games you have
gamelauncher_cli.exe -list
```

#### **Integration with Other Tools:**
You can now integrate the game launcher with:
- **Desktop shortcuts**: Create shortcuts with `-game 1` arguments
- **Task scheduler**: Automate game launches
- **Batch files**: Create custom launch scripts
- **Voice assistants**: Use voice commands to launch games

## 🔧 Technical Details

### **File Structure:**
- `gamelauncher.exe` - Main GUI application with CLI support
- `gamelauncher_cli.exe` - Pure command-line interface
- `gamelauncher_console.exe` - Interactive console application

### **Data Persistence:**
- Deleted games are permanently removed from `games.json`
- No backup is created automatically (consider backing up your data)
- The deletion is immediate and cannot be undone

### **Error Handling:**
- Invalid game numbers show available games
- Missing executables show appropriate error messages
- Network errors during launch are reported clearly

## 🎯 Use Cases

### **Power Users:**
- Create desktop shortcuts for favorite games
- Use keyboard shortcuts to launch games
- Integrate with automation tools

### **Streamers/Content Creators:**
- Quick game switching during streams
- Automated game launches for scheduled content
- Integration with streaming software

### **System Administrators:**
- Deploy games across multiple machines
- Create standardized game launch procedures
- Monitor game usage and updates

## 🚀 Future Enhancements

Potential improvements for future versions:
- **Batch operations**: Delete multiple games at once
- **Game categories**: Organize games by type/genre
- **Launch parameters**: Pass custom arguments to games
- **Scheduled launches**: Automatically launch games at specific times
- **Game statistics**: Track play time and launch frequency 