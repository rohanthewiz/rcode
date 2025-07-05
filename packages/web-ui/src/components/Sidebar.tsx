import { useEffect } from 'react';
import { Link, useParams, useNavigate } from 'react-router-dom';
import { format } from 'date-fns';
import { useSessionStore } from '@/store/session';
import { useChatStore } from '@/store/chat';
import clsx from 'clsx';

export function Sidebar() {
  const navigate = useNavigate();
  const { sessionId } = useParams();
  const {
    sessions,
    activeSessionId,
    loadSessions,
    createSession,
    deleteSession,
    setActiveSession,
  } = useSessionStore();
  const { clearMessages } = useChatStore();

  useEffect(() => {
    loadSessions().catch(console.error);
  }, []);

  useEffect(() => {
    if (sessionId && sessionId !== activeSessionId) {
      setActiveSession(sessionId);
    }
  }, [sessionId, activeSessionId]);

  const handleNewSession = async () => {
    try {
      const session = await createSession();
      navigate(`/chat/${session.id}`);
    } catch (error) {
      console.error('Failed to create session:', error);
    }
  };

  const handleDeleteSession = async (id: string, e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    
    try {
      await deleteSession(id);
      clearMessages(id);
      
      // If we deleted the active session, navigate to a different one
      if (id === activeSessionId) {
        const remainingSessions = sessions.filter(s => s.id !== id);
        if (remainingSessions.length > 0) {
          navigate(`/chat/${remainingSessions[0].id}`);
        } else {
          navigate('/chat');
        }
      }
    } catch (error) {
      console.error('Failed to delete session:', error);
    }
  };

  return (
    <aside className="w-64 bg-secondary/50 border-r border-border flex flex-col">
      {/* Header */}
      <div className="p-4 border-b border-border">
        <h1 className="text-xl font-semibold">OpenCode</h1>
      </div>

      {/* New Session Button */}
      <div className="p-3">
        <button
          onClick={handleNewSession}
          className="w-full px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors"
        >
          New Session
        </button>
      </div>

      {/* Session List */}
      <div className="flex-1 overflow-y-auto">
        <div className="px-3 py-2">
          <h2 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">
            Sessions
          </h2>
          <div className="space-y-1">
            {sessions.map((session) => (
              <Link
                key={session.id}
                to={`/chat/${session.id}`}
                className={clsx(
                  'block px-3 py-2 rounded-md text-sm truncate group relative',
                  'hover:bg-accent hover:text-accent-foreground transition-colors',
                  session.id === activeSessionId && 'bg-accent text-accent-foreground'
                )}
              >
                <div className="flex items-center justify-between">
                  <div className="flex-1 truncate">
                    <div className="truncate font-medium">
                      {session.title || 'Untitled Session'}
                    </div>
                    <div className="text-xs text-muted-foreground">
                      {format(new Date(session.time.updated), 'MMM d, h:mm a')}
                    </div>
                  </div>
                  <button
                    onClick={(e) => handleDeleteSession(session.id, e)}
                    className="opacity-0 group-hover:opacity-100 ml-2 p-1 hover:bg-secondary rounded transition-opacity"
                    aria-label="Delete session"
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </div>
                {session.share && (
                  <div className="flex items-center gap-1 mt-1">
                    <svg className="w-3 h-3 text-muted-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.684 13.342C8.886 12.938 9 12.482 9 12c0-.482-.114-.938-.316-1.342m0 2.684a3 3 0 110-2.684m9.632 4.268C18.114 15.938 18 16.482 18 17c0 1.657-1.343 3-3 3s-3-1.343-3-3 1.343-3 3-3c.482 0 .938.114 1.342.316m0 0a3 3 0 00-4.268-4.268m4.268 4.268a3 3 0 01-4.268 4.268" />
                    </svg>
                    <span className="text-xs text-muted-foreground">Shared</span>
                  </div>
                )}
              </Link>
            ))}
          </div>
        </div>
      </div>

      {/* Footer */}
      <div className="p-4 border-t border-border text-xs text-muted-foreground">
        <div>v{useAppStore.getState().appInfo?.version || '0.0.0'}</div>
      </div>
    </aside>
  );
}