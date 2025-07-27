/**
 * Browser-based test for RCode Diff Visualization
 * Copy and paste this into the browser console while RCode is running
 */

console.log('%c🧪 RCode Diff Visualization Test Suite', 'color: #4CAF50; font-size: 16px; font-weight: bold');

// Test Configuration
const TEST_CONFIG = {
    testFilePath: 'test/sample.js',
    testDiffId: 'test-diff-' + Date.now(),
    delays: {
        short: 100,
        medium: 500,
        long: 1000
    }
};

// Test Results Tracker
const testResults = {
    passed: 0,
    failed: 0,
    tests: []
};

// Test Helper Functions
function assert(condition, testName, errorMsg = '') {
    if (condition) {
        console.log(`✅ ${testName}`);
        testResults.passed++;
        testResults.tests.push({ name: testName, passed: true });
    } else {
        console.error(`❌ ${testName}${errorMsg ? ': ' + errorMsg : ''}`);
        testResults.failed++;
        testResults.tests.push({ name: testName, passed: false, error: errorMsg });
    }
}

function delay(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

// Test Suite
async function runDiffVisualizationTests() {
    console.group('📊 Component Initialization Tests');
    
    // Test 1: Check core components
    assert(typeof window.diffViewer !== 'undefined', 'DiffViewer is initialized');
    assert(typeof window.FileExplorer !== 'undefined', 'FileExplorer is initialized');
    assert(typeof window.currentSessionId !== 'undefined', 'Session ID is available');
    assert(typeof monaco !== 'undefined', 'Monaco Editor is loaded');
    
    console.groupEnd();
    
    console.group('🔄 SSE Event Handling Tests');
    
    // Test 2: Simulate diff_available event
    const mockDiffEvent = {
        type: 'diff_available',
        sessionId: window.currentSessionId,
        data: {
            diffId: TEST_CONFIG.testDiffId,
            path: TEST_CONFIG.testFilePath,
            stats: {
                additions: 10,
                deletions: 5
            }
        }
    };
    
    // Store original SSE handler
    const originalHandler = window.handleSSEMessage;
    let sseEventReceived = false;
    
    // Mock SSE handler to track event
    window.handleSSEMessage = function(event) {
        if (originalHandler) originalHandler.call(this, event);
        const data = JSON.parse(event.data);
        if (data.type === 'diff_available') {
            sseEventReceived = true;
        }
    };
    
    // Trigger mock event
    window.handleSSEMessage({ data: JSON.stringify(mockDiffEvent) });
    await delay(TEST_CONFIG.delays.short);
    
    assert(sseEventReceived, 'SSE diff_available event processed');
    
    // Restore original handler
    window.handleSSEMessage = originalHandler;
    
    console.groupEnd();
    
    console.group('📁 File Explorer Integration Tests');
    
    // Test 3: Check if file is marked as modified
    assert(
        window.FileExplorer.isFileModified(TEST_CONFIG.testFilePath),
        'File marked as modified in FileExplorer'
    );
    
    // Test 4: Check if diff is stored in viewer
    const storedDiffId = window.diffViewer.getLatestDiff(TEST_CONFIG.testFilePath);
    assert(
        storedDiffId === TEST_CONFIG.testDiffId,
        'Diff ID stored correctly',
        `Expected ${TEST_CONFIG.testDiffId}, got ${storedDiffId}`
    );
    
    console.groupEnd();
    
    console.group('🎨 UI Element Tests');
    
    // Test 5: Check for diff modal
    const diffModal = document.getElementById('diff-modal');
    assert(diffModal !== null, 'Diff modal element exists');
    
    // Test 6: Check for view mode buttons
    const viewModeButtons = document.querySelectorAll('.diff-mode');
    assert(viewModeButtons.length === 4, 'All view mode buttons present', 
        `Found ${viewModeButtons.length} buttons`);
    
    // Test 7: Check for diff container
    const diffContainer = document.getElementById('diff-container');
    assert(diffContainer !== null, 'Diff container element exists');
    
    console.groupEnd();
    
    console.group('🔧 Functional Tests');
    
    // Test 8: Test diff viewer show/hide
    const initialModalState = diffModal.classList.contains('active');
    
    // Create mock diff data
    window.diffViewer.currentDiff = {
        id: TEST_CONFIG.testDiffId,
        path: TEST_CONFIG.testFilePath,
        before: 'const x = 1;\nconst y = 2;',
        after: 'const x = 1;\nconst y = 3;\nconst z = 4;',
        stats: { additions: 2, deletions: 1 }
    };
    
    // Show diff viewer
    window.diffViewer.modal.classList.add('active');
    await delay(TEST_CONFIG.delays.short);
    
    assert(
        window.diffViewer.modal.classList.contains('active'),
        'Diff viewer can be shown'
    );
    
    // Test view mode switching
    let viewModeSwitchSuccess = true;
    const modes = ['monaco', 'side-by-side', 'inline', 'unified'];
    
    for (const mode of modes) {
        try {
            window.diffViewer.setViewMode(mode);
            await delay(TEST_CONFIG.delays.short);
            const activeButton = document.querySelector(`.diff-mode[data-mode="${mode}"]`);
            if (!activeButton || !activeButton.classList.contains('active')) {
                viewModeSwitchSuccess = false;
                break;
            }
        } catch (e) {
            viewModeSwitchSuccess = false;
            console.error(`Error switching to ${mode} mode:`, e);
            break;
        }
    }
    
    assert(viewModeSwitchSuccess, 'View mode switching works');
    
    // Hide diff viewer
    window.diffViewer.close();
    await delay(TEST_CONFIG.delays.short);
    
    assert(
        !window.diffViewer.modal.classList.contains('active'),
        'Diff viewer can be closed'
    );
    
    console.groupEnd();
    
    console.group('🧹 Cleanup Tests');
    
    // Test 9: Unmark file as modified
    window.FileExplorer.unmarkFileModified(TEST_CONFIG.testFilePath);
    assert(
        !window.FileExplorer.isFileModified(TEST_CONFIG.testFilePath),
        'File can be unmarked as modified'
    );
    
    console.groupEnd();
    
    // Display test summary
    console.log('\n' + '='.repeat(50));
    console.log('%c📊 Test Summary', 'font-size: 14px; font-weight: bold');
    console.log('='.repeat(50));
    console.log(`Total Tests: ${testResults.passed + testResults.failed}`);
    console.log(`%c✅ Passed: ${testResults.passed}`, 'color: #4CAF50');
    console.log(`%c❌ Failed: ${testResults.failed}`, 'color: #f44336');
    console.log('='.repeat(50));
    
    if (testResults.failed > 0) {
        console.group('Failed Tests Details');
        testResults.tests
            .filter(t => !t.passed)
            .forEach(t => console.error(`❌ ${t.name}: ${t.error || 'No error message'}`));
        console.groupEnd();
    }
    
    return testResults;
}

// Interactive Test Functions
window.rcodeTests = {
    // Run all tests
    runAll: runDiffVisualizationTests,
    
    // Create a test diff
    createTestDiff: function(path = 'test/example.js') {
        const testEvent = {
            type: 'diff_available',
            sessionId: window.currentSessionId,
            data: {
                diffId: 'manual-test-' + Date.now(),
                path: path,
                stats: { additions: 5, deletions: 2 }
            }
        };
        
        window.handleSSEMessage({ data: JSON.stringify(testEvent) });
        console.log(`📝 Created test diff for ${path}`);
        return testEvent.data.diffId;
    },
    
    // Show a test diff
    showTestDiff: function() {
        if (!window.diffViewer) {
            console.error('DiffViewer not initialized');
            return;
        }
        
        // Create mock diff
        window.diffViewer.currentDiff = {
            id: 'demo-diff',
            path: 'demo/example.js',
            before: `function calculateSum(a, b) {
    return a + b;
}

module.exports = { calculateSum };`,
            after: `function calculateSum(a, b) {
    // Add input validation
    if (typeof a !== 'number' || typeof b !== 'number') {
        throw new Error('Both arguments must be numbers');
    }
    return a + b;
}

function calculateProduct(a, b) {
    return a * b;
}

module.exports = { calculateSum, calculateProduct };`,
            stats: { additions: 8, deletions: 0 }
        };
        
        window.diffViewer.modal.classList.add('active');
        window.diffViewer.updateModalHeader();
        window.diffViewer.renderDiff();
        
        console.log('📊 Test diff viewer opened');
    },
    
    // Test synchronized scrolling
    testSyncScroll: async function() {
        console.log('🔄 Testing synchronized scrolling...');
        
        // Show test diff first
        this.showTestDiff();
        await delay(500);
        
        // Switch to side-by-side view
        window.diffViewer.setViewMode('side-by-side');
        await delay(500);
        
        const beforeContent = document.getElementById('diff-content-before');
        const afterContent = document.getElementById('diff-content-after');
        
        if (beforeContent && afterContent) {
            // Scroll the before content
            beforeContent.scrollTop = 100;
            await delay(100);
            
            if (Math.abs(afterContent.scrollTop - 100) < 5) {
                console.log('✅ Synchronized scrolling works!');
            } else {
                console.error('❌ Synchronized scrolling failed');
            }
        } else {
            console.error('❌ Could not find scroll containers');
        }
    },
    
    // Clean up all test data
    cleanup: function() {
        // Close diff viewer
        if (window.diffViewer) {
            window.diffViewer.close();
        }
        
        // Clear all modified file markers
        if (window.FileExplorer && window.FileExplorer.modifiedFiles) {
            window.FileExplorer.modifiedFiles.clear();
            window.FileExplorer.refreshTree();
        }
        
        console.log('🧹 Test cleanup complete');
    }
};

// Display usage instructions
console.log('\n%c🚀 Interactive Test Commands:', 'color: #2196F3; font-weight: bold');
console.log('  rcodeTests.runAll()        - Run all automated tests');
console.log('  rcodeTests.createTestDiff() - Create a test diff event');
console.log('  rcodeTests.showTestDiff()   - Open diff viewer with test data');
console.log('  rcodeTests.testSyncScroll() - Test synchronized scrolling');
console.log('  rcodeTests.cleanup()        - Clean up test data');
console.log('\nRun `rcodeTests.runAll()` to start the test suite.');

// Auto-run tests if requested
if (window.location.hash === '#test-diff') {
    console.log('\n🚀 Auto-running tests...\n');
    runDiffVisualizationTests();
}