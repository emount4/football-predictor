import { useCallback, useEffect, useMemo, useState } from "react";
import { ApiClient } from "./api";
import { getTelegramInitData, initTelegram } from "./telegram";
import type {
  AppPage,
  LeaderboardItem,
  MatchForUser,
  PredictionChoice,
  PredictionHistoryItem,
  UserProfile,
} from "./types";

const CHOICES: Array<{ value: PredictionChoice; label: string; hint: string }> = [
  { value: "1", label: "П1", hint: "победа хозяев" },
  { value: "X", label: "X", hint: "ничья" },
  { value: "2", label: "П2", hint: "победа гостей" },
];

function App() {
  const [page, setPage] = useState<AppPage>(() => getPageFromHash());
  const [me, setMe] = useState<UserProfile | null>(null);
  const [matches, setMatches] = useState<MatchForUser[]>([]);
  const [results, setResults] = useState<PredictionHistoryItem[]>([]);
  const [leaderboard, setLeaderboard] = useState<LeaderboardItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [actionMatchID, setActionMatchID] = useState<number | null>(null);
  const [error, setError] = useState("");

  const initData = useMemo(() => getTelegramInitData(), []);
  const api = useMemo(() => new ApiClient(initData), [initData]);

  const loadPage = useCallback(async () => {
    setError("");
    setLoading(true);

    try {
      const profile = await api.getMe();
      setMe(profile);

      if (page === "matches") {
        const data = await api.getMatches("active");
        setMatches(data);
      }

      if (page === "results") {
        const data = await api.getMyResults("finished");
        setResults(data);
      }

      if (page === "leaderboard") {
        const data = await api.getLeaderboard(100);
        setLeaderboard(data);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Неизвестная ошибка");
    } finally {
      setLoading(false);
    }
  }, [api, page]);

  useEffect(() => {
    initTelegram();

    const onHashChange = () => setPage(getPageFromHash());
    window.addEventListener("hashchange", onHashChange);

    if (!window.location.hash) {
      window.location.hash = "matches";
    }

    return () => window.removeEventListener("hashchange", onHashChange);
  }, []);

  useEffect(() => {
    void loadPage();
  }, [loadPage]);

  async function submitPrediction(matchID: number, choice: PredictionChoice) {
    setActionMatchID(matchID);
    setError("");

    try {
      await api.submitPrediction(matchID, choice);
      const [profile, nextMatches] = await Promise.all([api.getMe(), api.getMatches("active")]);
      setMe(profile);
      setMatches(nextMatches);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось сохранить прогноз");
    } finally {
      setActionMatchID(null);
    }
  }

  return (
    <main className="app-shell">
      <section className="hero-card">
        <div>
          <p className="eyebrow">Football Predictor</p>
          <h1>Прогнозы без шума</h1>
          <p className="subtitle">Выбирай исход матча, смотри результаты и держись в топе лидерборда.</p>
        </div>

        <ProfilePill me={me} />
      </section>

      <nav className="tabs" aria-label="Навигация">
        <TabButton page="matches" current={page} label="Матчи" />
        <TabButton page="results" current={page} label="Результаты" />
        <TabButton page="leaderboard" current={page} label="Лидерборд" />
      </nav>

      {error && <div className="alert">{error}</div>}

      {loading ? (
        <Skeleton />
      ) : (
        <>
          {page === "matches" && (
            <MatchesPage
              matches={matches}
              busyMatchID={actionMatchID}
              onSubmitPrediction={submitPrediction}
            />
          )}

          {page === "results" && <ResultsPage results={results} />}

          {page === "leaderboard" && <LeaderboardPage leaderboard={leaderboard} me={me} />}
        </>
      )}
    </main>
  );
}

function getPageFromHash(): AppPage {
  const hash = window.location.hash.replace("#", "");
  if (hash === "results" || hash === "leaderboard" || hash === "matches") {
    return hash;
  }
  return "matches";
}

function TabButton({ page, current, label }: { page: AppPage; current: AppPage; label: string }) {
  return (
    <button
      type="button"
      className={`tab ${page === current ? "tab-active" : ""}`}
      onClick={() => {
        window.location.hash = page;
      }}
    >
      {label}
    </button>
  );
}

function ProfilePill({ me }: { me: UserProfile | null }) {
  if (!me) {
    return <div className="profile-pill muted">Telegram</div>;
  }

  const title = me.display_name || me.username || "Игрок";

  return (
    <div className="profile-pill">
      {me.photo_url ? <img src={me.photo_url} alt="" /> : <div className="avatar-fallback">{title[0]}</div>}
      <div>
        <strong>{title}</strong>
        <span>{me.total_points} очк. · #{me.rank || "—"}</span>
      </div>
    </div>
  );
}

function MatchesPage({
  matches,
  busyMatchID,
  onSubmitPrediction,
}: {
  matches: MatchForUser[];
  busyMatchID: number | null;
  onSubmitPrediction: (matchID: number, choice: PredictionChoice) => Promise<void>;
}) {
  if (matches.length === 0) {
    return <EmptyState title="Матчей пока нет" text="Сервер еще не загрузил ближайшее расписание или сегодня нет игр." />;
  }

  return (
    <section className="grid-list">
      {matches.map((match) => (
        <article className="match-card" key={match.api_id}>
          <MatchHeader match={match} />

          <div className="teams">
            <TeamName name={match.home_team} />
            <div className="versus">vs</div>
            <TeamName name={match.away_team} />
          </div>

          <ChoiceRow
            selected={match.my_prediction}
            locked={match.prediction_locked || match.status !== "SCHEDULED"}
            loading={busyMatchID === match.api_id}
            onPick={(choice) => onSubmitPrediction(match.api_id, choice)}
          />
        </article>
      ))}
    </section>
  );
}

function ResultsPage({ results }: { results: PredictionHistoryItem[] }) {
  if (results.length === 0) {
    return <EmptyState title="Результатов пока нет" text="Завершенные прогнозы появятся здесь после окончания матчей." />;
  }

  return (
    <section className="grid-list">
      {results.map((item) => (
        <article className="match-card" key={`${item.match_id}-${item.user_choice}`}>
          <div className="card-topline">
            <span>{item.league_name}</span>
            <StatusBadge status={item.status} />
          </div>

          <div className="score-row">
            <span>{item.home_team}</span>
            <strong>
              {formatScore(item.home_goals, item.away_goals)}
            </strong>
            <span>{item.away_team}</span>
          </div>

          <div className="result-footer">
            <span>
              Твой прогноз: <strong>{choiceLabel(item.user_choice)}</strong>
            </span>
            <span className={item.is_correct ? "result-ok" : "result-bad"}>
              {item.is_correct ? `+${item.points_awarded} очко` : "0 очков"}
            </span>
          </div>
        </article>
      ))}
    </section>
  );
}

function LeaderboardPage({ leaderboard, me }: { leaderboard: LeaderboardItem[]; me: UserProfile | null }) {
  if (leaderboard.length === 0) {
    return <EmptyState title="Лидерборд пуст" text="Рейтинг появится после первых прогнозов." />;
  }

  return (
    <section className="leaderboard-card">
      {leaderboard.map((item) => {
        const name = item.display_name || item.username || "Игрок";
        const isMe = me && item.username && item.username === me.username;

        return (
          <div className={`leader-row ${isMe ? "leader-row-me" : ""}`} key={`${item.rank}-${item.username}-${name}`}>
            <div className="rank">#{item.rank}</div>
            <div className="leader-user">
              {item.photo_url ? <img src={item.photo_url} alt="" /> : <div className="avatar-fallback">{name[0]}</div>}
              <div>
                <strong>{name}</strong>
                {item.username && <span>@{item.username}</span>}
              </div>
            </div>
            <div className="points">{item.total_points}</div>
          </div>
        );
      })}
    </section>
  );
}

function MatchHeader({ match }: { match: MatchForUser }) {
  return (
    <div className="card-topline">
      <span>{match.league_name}</span>
      <div className="badges">
        <span className="date-pill">{formatDate(match.match_time)}</span>
        <StatusBadge status={match.status} />
      </div>
    </div>
  );
}

function TeamName({ name }: { name: string }) {
  return <div className="team-name">{name}</div>;
}

function ChoiceRow({
  selected,
  locked,
  loading,
  onPick,
}: {
  selected: PredictionChoice | null;
  locked: boolean;
  loading: boolean;
  onPick: (choice: PredictionChoice) => void;
}) {
  return (
    <div className="choice-panel">
      <div className="choice-label">
        {locked ? "Прогноз закрыт" : selected ? "Твой прогноз сохранен" : "Выбери исход"}
      </div>

      <div className="choice-row">
        {CHOICES.map((choice) => (
          <button
            key={choice.value}
            type="button"
            className={`choice-button ${selected === choice.value ? "choice-selected" : ""}`}
            disabled={locked || loading}
            onClick={() => onPick(choice.value)}
            title={choice.hint}
          >
            {loading ? "..." : choice.label}
          </button>
        ))}
      </div>
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  const normalized = status.toUpperCase();

  const label =
    normalized === "SCHEDULED"
      ? "скоро"
      : normalized === "LIVE"
        ? "live"
        : normalized === "FINISHED"
          ? "завершен"
          : normalized.toLowerCase();

  return <span className={`status status-${normalized.toLowerCase()}`}>{label}</span>;
}

function EmptyState({ title, text }: { title: string; text: string }) {
  return (
    <section className="empty-state">
      <div className="empty-icon">⚽</div>
      <h2>{title}</h2>
      <p>{text}</p>
    </section>
  );
}

function Skeleton() {
  return (
    <section className="grid-list">
      <div className="skeleton-card" />
      <div className="skeleton-card" />
      <div className="skeleton-card" />
    </section>
  );
}

function formatDate(value: string): string {
  return new Intl.DateTimeFormat("ru-RU", {
    day: "2-digit",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}

function formatScore(home: number | null, away: number | null): string {
  if (home === null || away === null) {
    return "— : —";
  }
  return `${home} : ${away}`;
}

function choiceLabel(choice: PredictionChoice): string {
  if (choice === "1") return "П1";
  if (choice === "2") return "П2";
  return "X";
}

export default App;
