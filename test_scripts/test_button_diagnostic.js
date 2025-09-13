// Diagnostic script to check permission button functionality
// Run this in the browser console at http://localhost:8000

console.log('=== Permission Dialog Button Diagnostic ===');

// 1. Check if PermissionsModule exists
console.log('1. PermissionsModule exists?', !!window.PermissionsModule);

// 2. Check if buttons exist in DOM
const approveBtn = document.getElementById('permission-approve');
const denyBtn = document.getElementById('permission-deny'); 
const abortBtn = document.getElementById('permission-abort');

console.log('2. Buttons in DOM:');
console.log('   - Approve button:', !!approveBtn);
console.log('   - Deny button:', !!denyBtn);
console.log('   - Abort button:', !!abortBtn);

// 3. Check event listeners on buttons
if (approveBtn) {
    const listeners = getEventListeners ? getEventListeners(approveBtn) : 'getEventListeners not available';
    console.log('3. Approve button event listeners:', listeners);
}

// 4. Try to manually trigger a test permission request
console.log('4. Triggering test permission request...');

const testRequest = {
    sessionId: window.currentSessionId || 'test-session',
    data: {
        toolName: 'write_file',
        parameterDisplay: 'Test file: /tmp/test.txt',
        parameters: {
            path: '/tmp/test.txt',
            content: 'Test content'
        },
        requestId: 'test-' + Date.now(),
        timestamp: Date.now()
    }
};

try {
    if (window.PermissionsModule) {
        console.log('Calling PermissionsModule.handlePermissionRequest with:', testRequest);
        window.PermissionsModule.handlePermissionRequest(testRequest);
        
        // Check if modal is visible
        setTimeout(() => {
            const modal = document.getElementById('permission-modal');
            console.log('5. Modal visible after request?', modal && modal.style.display !== 'none');
            
            // Check button state after modal shows
            const newApproveBtn = document.getElementById('permission-approve');
            if (newApproveBtn) {
                // Try clicking the button programmatically
                console.log('6. Attempting programmatic click on Approve button...');
                newApproveBtn.click();
            }
        }, 100);
    }
} catch (error) {
    console.error('Error during test:', error);
}

// 5. Add temporary click handlers to see if buttons are responsive
console.log('7. Adding diagnostic click handlers...');
if (approveBtn) {
    approveBtn.addEventListener('click', () => {
        console.log('DIAGNOSTIC: Approve button was clicked!');
    });
}
if (denyBtn) {
    denyBtn.addEventListener('click', () => {
        console.log('DIAGNOSTIC: Deny button was clicked!');
    });
}
if (abortBtn) {
    abortBtn.addEventListener('click', () => {
        console.log('DIAGNOSTIC: Abort button was clicked!');
    });
}

console.log('=== Diagnostic complete. Try clicking the buttons now. ===');