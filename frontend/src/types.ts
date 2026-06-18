export type AppPage = "matches" | "results" | "leaderboard";

export type MatchStatus = "SCHEDULED" | "LIVE" | "FINISHED" | "CANCELLED" | string;
export type PredictionChoice = "1" | "X" | "2";

export interface UserProfile {
  username: string;
  display_name: string;
  photo_url: string;
  total_points: number;
  rank: number;
  predictions_count: number;
  correct_predictions: number;
}

export interface MatchForUser {
  api_id: number;
  home_team: string;
  away_team: string;
  league_id: number;
  league_name: string;
  match_time: string;
  status: MatchStatus;
  home_goals: number | null;
  away_goals: number | null;
  outcome: string;
  my_prediction: PredictionChoice | null;
  prediction_locked: boolean;
}

export interface MatchesPage {
  items: MatchForUser[];
  total: number;
  page: number;
  limit: number;
}

export interface PredictionHistoryItem {
  match_id: number;
  home_team: string;
  away_team: string;
  league_id: number;
  league_name: string;
  match_time: string;
  status: MatchStatus;
  home_goals: number | null;
  away_goals: number | null;
  outcome: string;
  user_choice: PredictionChoice;
  is_correct: boolean | null;
  points_awarded: number;
}

export interface LeaderboardItem {
  rank: number;
  username: string;
  display_name: string;
  photo_url: string;
  total_points: number;
}

export interface ApiErrorBody {
  error?: string;
  message?: string;
}

export interface PredictionVoter {
  username: string;
  display_name: string;
  photo_url: string;
  user_choice: PredictionChoice;
}

export interface PredictionStats {
  match_id: number;
  total: number;
  choices: Record<PredictionChoice, number>;
  percent: Record<PredictionChoice, number>;
  voters: PredictionVoter[];
}
