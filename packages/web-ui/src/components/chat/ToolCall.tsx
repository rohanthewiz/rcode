import { useState } from 'react';
import type { ToolInvocation } from '@/types/api';
import clsx from 'clsx';

interface ToolCallProps {
  invocation: ToolInvocation;
  metadata?: any;
  isExpanded: boolean;
  onToggle: () => void;
}

export function ToolCall({ invocation, metadata, isExpanded, onToggle }: ToolCallProps) {
  const isLoading = invocation.state === 'partial-call' || invocation.state === 'call';
  const isError = metadata?.error;
  const duration = metadata?.time ? metadata.time.end - metadata.time.start : 0;

  const getToolIcon = (toolName: string) => {
    switch (toolName) {
      case 'bash':
        return '🖥️';
      case 'read':
        return '📄';
      case 'write':
        return '✏️';
      case 'edit':
      case 'multiedit':
        return '📝';
      case 'grep':
      case 'glob':
        return '🔍';
      case 'ls':
        return '📁';
      case 'webfetch':
        return '🌐';
      default:
        return '🔧';
    }
  };

  const formatDuration = (ms: number) => {
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(1)}s`;
  };

  return (
    <div className={clsx(
      'border rounded-md overflow-hidden',
      isError ? 'border-red-500' : 'border-border'
    )}>
      {/* Header */}
      <button
        onClick={onToggle}
        className={clsx(
          'w-full px-3 py-2 flex items-center justify-between text-sm hover:bg-accent transition-colors',
          isError ? 'bg-red-50 dark:bg-red-900/20' : 'bg-secondary/50'
        )}
      >
        <div className="flex items-center gap-2">
          <span className="text-lg">{getToolIcon(invocation.toolName)}</span>
          <span className="font-medium">{invocation.toolName}</span>
          {metadata?.title && (
            <span className="text-muted-foreground">{metadata.title}</span>
          )}
        </div>
        
        <div className="flex items-center gap-2">
          {isLoading && (
            <div className="w-4 h-4 border-2 border-primary border-t-transparent rounded-full animate-spin" />
          )}
          {duration > 0 && (
            <span className="text-xs text-muted-foreground">{formatDuration(duration)}</span>
          )}
          <svg 
            className={clsx(
              'w-4 h-4 transition-transform',
              isExpanded && 'rotate-180'
            )} 
            fill="none" 
            stroke="currentColor" 
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </div>
      </button>

      {/* Content */}
      {isExpanded && (
        <div className="p-3 border-t border-border">
          {/* Arguments */}
          {invocation.args && Object.keys(invocation.args).length > 0 && (
            <div className="mb-3">
              <h4 className="text-xs font-medium text-muted-foreground mb-1">Arguments</h4>
              <pre className="text-xs bg-secondary p-2 rounded overflow-x-auto">
                {JSON.stringify(invocation.args, null, 2)}
              </pre>
            </div>
          )}

          {/* Result */}
          {invocation.state === 'result' && invocation.result && (
            <div>
              <h4 className="text-xs font-medium text-muted-foreground mb-1">Result</h4>
              <pre className="text-xs bg-secondary p-2 rounded overflow-x-auto whitespace-pre-wrap">
                {invocation.result}
              </pre>
            </div>
          )}

          {/* Error */}
          {isError && metadata?.message && (
            <div className="text-sm text-red-600 dark:text-red-400">
              {metadata.message}
            </div>
          )}
        </div>
      )}
    </div>
  );
}