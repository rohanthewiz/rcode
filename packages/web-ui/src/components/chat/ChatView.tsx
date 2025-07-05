import { useEffect, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useSessionStore } from '@/store/session';
import { useChatStore } from '@/store/chat';
import { MessageList } from './MessageList';
import { ChatInput } from './ChatInput';

export function ChatView() {
  const { sessionId } = useParams();
  const navigate = useNavigate();
  const { sessions, createSession } = useSessionStore();
  const { messages, loadMessages, isStreaming } = useChatStore();
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Create a session if none exists
  useEffect(() => {
    if (!sessionId && sessions.length === 0) {
      createSession().then(session => {
        navigate(`/chat/${session.id}`, { replace: true });
      });
    } else if (!sessionId && sessions.length > 0) {
      // Navigate to the most recent session
      navigate(`/chat/${sessions[0].id}`, { replace: true });
    }
  }, [sessionId, sessions]);

  // Load messages when session changes
  useEffect(() => {
    if (sessionId) {
      loadMessages(sessionId).catch(console.error);
    }
  }, [sessionId]);

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages[sessionId || '']]);

  if (!sessionId) {
    return null;
  }

  const sessionMessages = messages[sessionId] || [];
  const isSessionStreaming = isStreaming[sessionId] || false;

  return (
    <div className="flex flex-col h-full">
      {/* Messages */}
      <div className="flex-1 overflow-y-auto">
        <MessageList messages={sessionMessages} />
        <div ref={messagesEndRef} />
      </div>

      {/* Input */}
      <div className="border-t border-border">
        <ChatInput 
          sessionId={sessionId} 
          disabled={isSessionStreaming}
        />
      </div>
    </div>
  );
}