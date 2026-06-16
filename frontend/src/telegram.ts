declare global {
  interface Window {
    Telegram?: {
      WebApp?: {
        initData: string;
        ready: () => void;
        expand: () => void;
        close: () => void;
        colorScheme?: "light" | "dark";
      };
    };
  }
}

export function initTelegram(): void {
  const tg = window.Telegram?.WebApp;
  tg?.ready();
  tg?.expand();
}

export function getTelegramInitData(): string {
  const realInitData = window.Telegram?.WebApp?.initData;
  if (realInitData && realInitData.trim() !== "") {
    return realInitData;
  }

  return import.meta.env.VITE_DEV_INIT_DATA || "";
}
