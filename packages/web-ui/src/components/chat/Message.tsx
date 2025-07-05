import { useState } from 'react';
import ReactMarkdown from 'react-markdown';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { oneDark } from 'react-syntax-highlighter/dist/esm/styles/prism';
import remarkGfm from 'remark-gfm';
import type { MessageInfo, TextPart, FilePart, ToolInvocationPart } from '@/types/api';
import { ToolCall } from './ToolCall';
import clsx from 'clsx';

interface MessageProps {
  message: MessageInfo;
}

export function Message({ message }: MessageProps) {
  const isUser = message.role === 'user';
  const isError = !!message.metadata.error;
  const [expandedTools, setExpandedTools] = useState<Set<string>>(new Set());

  const toggleTool = (toolId: string) => {
    setExpandedTools(prev => {
      const next = new Set(prev);
      if (next.has(toolId)) {
        next.delete(toolId);
      } else {
        next.add(toolId);
      }
      return next;
    });
  };

  const renderPart = (part: MessageInfo['parts'][0], index: number) => {
    switch (part.type) {
      case 'text':
        const textPart = part as TextPart;
        if (!textPart.text) return null;
        
        if (isUser) {
          return (
            <div key={index} className="prose prose-sm max-w-none dark:prose-invert">
              {textPart.text}
            </div>
          );
        }
        
        return (
          <ReactMarkdown
            key={index}
            className="prose prose-sm max-w-none dark:prose-invert"
            remarkPlugins={[remarkGfm]}
            components={{
              code({ node, inline, className, children, ...props }) {
                const match = /language-(\w+)/.exec(className || '');
                return !inline && match ? (
                  <SyntaxHighlighter
                    style={oneDark}
                    language={match[1]}
                    PreTag="div"
                    {...props}
                  >
                    {String(children).replace(/\n$/, '')}
                  </SyntaxHighlighter>
                ) : (
                  <code className={className} {...props}>
                    {children}
                  </code>
                );
              },
            }}
          >
            {textPart.text}
          </ReactMarkdown>
        );

      case 'file':
        const filePart = part as FilePart;
        return (
          <div key={index} className="flex items-center gap-2 p-2 bg-secondary rounded-md">
            <svg className="w-4 h-4 text-muted-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
            <span className="text-sm">{filePart.filename || 'File'}</span>
            <span className="text-xs text-muted-foreground">({filePart.mediaType})</span>
          </div>
        );

      case 'tool-invocation':
        const toolPart = part as ToolInvocationPart;
        return (
          <ToolCall
            key={index}
            invocation={toolPart.toolInvocation}
            metadata={message.metadata.tool[toolPart.toolInvocation.toolCallId]}
            isExpanded={expandedTools.has(toolPart.toolInvocation.toolCallId)}
            onToggle={() => toggleTool(toolPart.toolInvocation.toolCallId)}
          />
        );

      case 'step-start':
        return (
          <div key={index} className="flex items-center gap-2 text-sm text-muted-foreground">
            <div className="w-2 h-2 bg-primary rounded-full animate-pulse" />
            <span>Thinking...</span>
          </div>
        );

      default:
        return null;
    }
  };

  return (
    <div className={clsx(
      'flex gap-3',
      isUser ? 'justify-end' : 'justify-start'
    )}>
      {/* Avatar */}
      <div className={clsx(
        'flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium',
        isUser ? 'bg-primary text-primary-foreground order-2' : 'bg-secondary order-1'
      )}>
        {isUser ? 'U' : 'A'}
      </div>

      {/* Content */}
      <div className={clsx(
        'flex-1 max-w-2xl',
        isUser && 'flex flex-col items-end'
      )}>
        {/* Message parts */}
        <div className={clsx(
          'space-y-2',
          isUser && 'text-right'
        )}>
          {message.parts.map((part, index) => renderPart(part, index))}
        </div>

        {/* Error display */}
        {isError && (
          <div className="mt-2 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
            <div className="flex items-center gap-2 text-red-600 dark:text-red-400">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <span className="text-sm font-medium">Error</span>
            </div>
            <p className="mt-1 text-sm text-red-600 dark:text-red-400">
              {message.metadata.error?.data?.message || 'An error occurred'}
            </p>
          </div>
        )}

        {/* Metadata */}
        {message.metadata.assistant && (
          <div className="mt-2 flex items-center gap-4 text-xs text-muted-foreground">
            <span>{message.metadata.assistant.providerID} / {message.metadata.assistant.modelID}</span>
            {message.metadata.assistant.cost > 0 && (
              <span>${message.metadata.assistant.cost.toFixed(4)}</span>
            )}
            {message.metadata.assistant.tokens.input > 0 && (
              <span>
                {message.metadata.assistant.tokens.input + message.metadata.assistant.tokens.output} tokens
              </span>
            )}
          </div>
        )}
      </div>
    </div>
  );
}