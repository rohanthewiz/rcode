import { useEffect, useState } from 'react';
import { useAppStore } from '@/store/app';
import { useChatStore } from '@/store/chat';
import clsx from 'clsx';

export function ProviderSelector() {
  const { providers, defaultModels } = useAppStore();
  const { selectedProvider, selectedModel, setProvider } = useChatStore();
  const [isOpen, setIsOpen] = useState(false);

  // Initialize default provider/model
  useEffect(() => {
    if (!selectedProvider && providers.length > 0) {
      const defaultProvider = providers.find(p => p.id === 'anthropic') || providers[0];
      const defaultModel = defaultModels[defaultProvider.id] || Object.keys(defaultProvider.models)[0];
      setProvider(defaultProvider.id, defaultModel);
    }
  }, [providers, defaultModels, selectedProvider]);

  const currentProvider = providers.find(p => p.id === selectedProvider);
  const currentModel = currentProvider?.models[selectedModel || ''];

  const handleSelect = (providerId: string, modelId: string) => {
    setProvider(providerId, modelId);
    setIsOpen(false);
  };

  if (!currentProvider || !currentModel) {
    return null;
  }

  return (
    <div className="relative">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center gap-2 px-3 py-1.5 text-sm border border-border rounded-md hover:bg-accent transition-colors"
      >
        <span className="text-lg">{currentProvider.icon}</span>
        <span className="font-medium">{currentModel.name}</span>
        <svg className="w-4 h-4 text-muted-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {isOpen && (
        <>
          <div
            className="fixed inset-0 z-10"
            onClick={() => setIsOpen(false)}
          />
          <div className="absolute right-0 mt-2 w-80 max-h-96 overflow-y-auto bg-background border border-border rounded-lg shadow-lg z-20">
            {providers.map((provider) => (
              <div key={provider.id} className="p-2">
                <div className="flex items-center gap-2 px-2 py-1 text-sm font-medium text-muted-foreground">
                  <span className="text-lg">{provider.icon}</span>
                  <span>{provider.name}</span>
                </div>
                <div className="mt-1 space-y-1">
                  {Object.values(provider.models).map((model) => (
                    <button
                      key={model.id}
                      onClick={() => handleSelect(provider.id, model.id)}
                      className={clsx(
                        'w-full text-left px-3 py-2 text-sm rounded-md transition-colors',
                        'hover:bg-accent hover:text-accent-foreground',
                        provider.id === selectedProvider && model.id === selectedModel &&
                        'bg-accent text-accent-foreground'
                      )}
                    >
                      <div className="font-medium">{model.name}</div>
                      {model.description && (
                        <div className="text-xs text-muted-foreground mt-0.5">
                          {model.description}
                        </div>
                      )}
                      <div className="flex items-center gap-4 mt-1 text-xs text-muted-foreground">
                        {model.limit.context && (
                          <span>{(model.limit.context / 1000).toFixed(0)}k context</span>
                        )}
                        {model.cost.input > 0 && (
                          <span>${model.cost.input.toFixed(2)}/1M in</span>
                        )}
                        {model.cost.output > 0 && (
                          <span>${model.cost.output.toFixed(2)}/1M out</span>
                        )}
                      </div>
                    </button>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );
}