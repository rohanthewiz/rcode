import { useState } from 'react';
import QRCode from 'qrcode';
import { useSessionStore } from '@/store/session';
import type { SessionInfo } from '@/types/api';
import clsx from 'clsx';

interface ShareButtonProps {
  session: SessionInfo;
}

export function ShareButton({ session }: ShareButtonProps) {
  const { shareSession, unshareSession } = useSessionStore();
  const [isOpen, setIsOpen] = useState(false);
  const [qrCode, setQrCode] = useState<string>('');
  const [isLoading, setIsLoading] = useState(false);

  const handleShare = async () => {
    if (session.share) {
      setIsOpen(true);
      // Generate QR code if not already generated
      if (!qrCode && session.share.url) {
        const code = await QRCode.toDataURL(session.share.url);
        setQrCode(code);
      }
    } else {
      setIsLoading(true);
      try {
        await shareSession(session.id);
        setIsOpen(true);
      } catch (error) {
        console.error('Failed to share session:', error);
      } finally {
        setIsLoading(false);
      }
    }
  };

  const handleUnshare = async () => {
    setIsLoading(true);
    try {
      await unshareSession(session.id);
      setIsOpen(false);
      setQrCode('');
    } catch (error) {
      console.error('Failed to unshare session:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleCopyLink = () => {
    if (session.share?.url) {
      navigator.clipboard.writeText(session.share.url);
    }
  };

  return (
    <>
      <button
        onClick={handleShare}
        disabled={isLoading}
        className={clsx(
          'flex items-center gap-2 px-3 py-1.5 text-sm border rounded-md transition-colors',
          session.share
            ? 'border-primary text-primary hover:bg-primary hover:text-primary-foreground'
            : 'border-border hover:bg-accent',
          isLoading && 'opacity-50 cursor-not-allowed'
        )}
      >
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.684 13.342C8.886 12.938 9 12.482 9 12c0-.482-.114-.938-.316-1.342m0 2.684a3 3 0 110-2.684m9.632 4.268C18.114 15.938 18 16.482 18 17c0 1.657-1.343 3-3 3s-3-1.343-3-3 1.343-3 3-3c.482 0 .938.114 1.342.316m0 0a3 3 0 00-4.268-4.268m4.268 4.268a3 3 0 01-4.268 4.268" />
        </svg>
        <span>{session.share ? 'Shared' : 'Share'}</span>
      </button>

      {isOpen && session.share && (
        <>
          <div
            className="fixed inset-0 bg-black/50 z-40"
            onClick={() => setIsOpen(false)}
          />
          <div className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-background border border-border rounded-lg shadow-xl z-50 p-6 max-w-md w-full">
            <h3 className="text-lg font-semibold mb-4">Share Session</h3>
            
            {/* QR Code */}
            {qrCode && (
              <div className="flex justify-center mb-4">
                <img src={qrCode} alt="QR Code" className="w-48 h-48" />
              </div>
            )}

            {/* Share URL */}
            <div className="mb-4">
              <label className="block text-sm font-medium mb-2">Share URL</label>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={session.share.url}
                  readOnly
                  className="flex-1 px-3 py-2 text-sm bg-secondary border border-border rounded-md"
                />
                <button
                  onClick={handleCopyLink}
                  className="px-3 py-2 text-sm bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
                >
                  Copy
                </button>
              </div>
            </div>

            {/* Actions */}
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setIsOpen(false)}
                className="px-4 py-2 text-sm border border-border rounded-md hover:bg-accent"
              >
                Close
              </button>
              <button
                onClick={handleUnshare}
                disabled={isLoading}
                className="px-4 py-2 text-sm bg-red-600 text-white rounded-md hover:bg-red-700 disabled:opacity-50"
              >
                Unshare
              </button>
            </div>
          </div>
        </>
      )}
    </>
  );
}