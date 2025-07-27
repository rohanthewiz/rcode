# Testing the Permission System

## Test Setup

1. Start the RCode server (already running)
2. Open http://localhost:8000 in a browser
3. Create a new session or use an existing one

## Test Scenarios

### Test 1: Basic Permission Request
1. In the RCode chat, type: "Create a file called test_permission.txt with the content 'Hello Permission System!'"
2. The permission dialog should appear:
   - Shows "write_file" as the tool name
   - Shows the file path parameter
   - Has Approve/Deny buttons
   - Shows a 30-second countdown timer
3. Click "Approve"
4. The file should be created successfully

### Test 2: Deny Permission
1. Type: "Delete the file test_permission.txt"
2. When the permission dialog appears, click "Deny"
3. The tool should not execute
4. You should see an error message that the tool was not approved

### Test 3: Remember Choice
1. Type: "Create another file called test2.txt"
2. Check the "Remember this choice for this session" checkbox
3. Click "Approve"
4. Type: "Create a third file called test3.txt"
5. The permission dialog should NOT appear (remembered the choice)
6. The file should be created automatically

### Test 4: Timeout Handling
1. Type: "Run the command: ls -la"
2. When the permission dialog appears, wait for 30 seconds
3. The dialog should disappear
4. You should see a timeout message

### Test 5: Multiple Tools
1. Type: "Create a directory called test_dir and then create a file inside it called hello.txt"
2. You should get permission requests for both tools
3. Approve both to see the operations complete

## Expected Behaviors

- Permission dialogs appear for tools with "ask" permission mode
- Countdown timer updates every second
- Approve/Deny buttons work correctly
- Remember choice persists for the session
- Timeout is handled gracefully
- Multiple permission requests can be handled

## Checking Logs

Monitor the server logs for:
```
INFO[xxxx] Tool requires permission
INFO[xxxx] Created permission request
INFO[xxxx] Broadcasting SSE event: type=permission_request
INFO[xxxx] Permission response processed
```

## Notes

- By default, most tools start with "ask" permission mode
- You can change tool permissions in the Tools tab
- Remembered choices only apply to the current session