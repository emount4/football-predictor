import type {
  ApiErrorBody,
  LeaderboardItem,
  MatchForUser,
  MatchesPage,
  PredictionChoice,
  PredictionHistoryItem,
  PredictionStats,
  UserProfile,
} from "./types";

const rawApiBaseURL = (import.meta.env.VITE_API_BASE_URL || "").trim();
const API_BASE_URL = rawApiBaseURL ? rawApiBaseURL.replace(/\/$/, "") : "";

export class ApiClient {
  private readonly initData: string;

  constructor(initData: string) {
    this.initData = initData;
  }

  async getMe(): Promise<UserProfile> {
    return this.request<UserProfile>("/api/me");
  }

  async getMatches(
    status: "active" | "scheduled" | "live" | "finished" | "all" = "active",
    page = 1,
    limit = 20,
  ): Promise<MatchesPage> {
    const params = new URLSearchParams({
      status,
      page: String(page),
      limit: String(limit),
    });
    return this.request<MatchesPage>(`/api/matches?${params.toString()}`);
  }

  async getMyResults(status: "finished" | "all" = "finished"): Promise<PredictionHistoryItem[]> {
    return this.request<PredictionHistoryItem[]>(`/api/predictions/me?status=${encodeURIComponent(status)}`);
  }

  async getLeaderboard(limit = 100): Promise<LeaderboardItem[]> {
    return this.request<LeaderboardItem[]>(`/api/leaderboard?limit=${limit}`);
  }

  async submitPrediction(matchID: number, userChoice: PredictionChoice): Promise<void> {
    await this.request("/api/predictions", {
      method: "POST",
      body: JSON.stringify({
        match_id: matchID,
        user_choice: userChoice,
      }),
    });
  }

  async getPredictionStats(matchID: number): Promise<PredictionStats> {
    return this.request<PredictionStats>(`/api/matches/${matchID}/prediction-stats`);
  }

  private async request<T = unknown>(path: string, init: RequestInit = {}): Promise<T> {
    if (!this.initData) {
      throw new Error("Нет Telegram initData. Открой приложение через Telegram Mini App.");
    }

    const response = await fetch(`${API_BASE_URL}${path}`, {
      ...init,
      headers: {
        "Content-Type": "application/json",
        Authorization: this.initData,
        ...(init.headers || {}),
      },
    });

    if (!response.ok) {
      let message = `HTTP ${response.status}`;

      try {
        const body = (await response.json()) as ApiErrorBody;
        message = body.error || body.message || message;
      } catch {
        // Тело не JSON — оставляем HTTP status.
      }

      throw new Error(message);
    }

    if (response.status === 204) {
      return undefined as T;
    }

    return (await response.json()) as T;
  }
}
