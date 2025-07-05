import { useSessionStore } from '@/store/session';
import { ProviderSelector } from './provider/ProviderSelector';
import { ShareButton } from './session/ShareButton';

export function Header() {
  const activeSession = useSessionStore(state => state.activeSession);

  return (
    <header className="h-14 border-b border-border px-4 flex items-center justify-between bg-background">
      <div className="flex items-center gap-4">
        {/* Session Title */}
        <h2 className="text-sm font-medium truncate max-w-xs">
          {activeSession?.title || 'Select a session'}
        </h2>
      </div>

      <div className="flex items-center gap-3">
        {/* Provider Selector */}
        <ProviderSelector />

        {/* Share Button */}
        {activeSession && <ShareButton session={activeSession} />}
      </div>
    </header>
  );
}