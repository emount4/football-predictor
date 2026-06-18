import { useCallback, useEffect, useMemo, useState } from "react";
import { ApiClient } from "./api";
import { getTelegramInitData, initTelegram } from "./telegram";
import type {
  AppPage,
  LeaderboardItem,
  MatchForUser,
  PredictionChoice,
  PredictionStats,
  PredictionVoter,
  UserProfile,
} from "./types";

const CHOICES: Array<{ value: PredictionChoice; label: string; hint: string }> = [
  { value: "1", label: "П1", hint: "победа хозяев" },
  { value: "X", label: "X", hint: "ничья" },
  { value: "2", label: "П2", hint: "победа гостей" },
];

const MATCHES_PAGE_SIZE = 20;

function App() {
  const [page, setPage] = useState<AppPage>(() => getPageFromHash());
  const [me, setMe] = useState<UserProfile | null>(null);
  const [matches, setMatches] = useState<MatchForUser[]>([]);
  const [matchesTotal, setMatchesTotal] = useState(0);
  const [matchesPage, setMatchesPage] = useState(1);
  const [results, setResults] = useState<MatchForUser[]>([]);
  const [resultsTotal, setResultsTotal] = useState(0);
  const [resultsPage, setResultsPage] = useState(1);
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
        const data = await api.getMatches("active", matchesPage, MATCHES_PAGE_SIZE);
        setMatches(data.items);
        setMatchesTotal(data.total);
      }

      if (page === "results") {
        const data = await api.getMatches("finished", resultsPage, MATCHES_PAGE_SIZE);
        setResults(data.items);
        setResultsTotal(data.total);
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
  }, [api, page, matchesPage, resultsPage]);

  useEffect(() => {
    initTelegram();

    const onHashChange = () => {
      setPage(getPageFromHash());
      setMatchesPage(1);
      setResultsPage(1);
    };
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
      const [profile, nextMatches] = await Promise.all([
        api.getMe(),
        api.getMatches("active", matchesPage, MATCHES_PAGE_SIZE),
      ]);
      setMe(profile);
      setMatches(nextMatches.items);
      setMatchesTotal(nextMatches.total);
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
          <h1>Прогнозы</h1>
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
              total={matchesTotal}
              currentPage={matchesPage}
              pageSize={MATCHES_PAGE_SIZE}
              onPageChange={setMatchesPage}
              busyMatchID={actionMatchID}
              api={api}
              onSubmitPrediction={submitPrediction}
            />
          )}

          {page === "results" && (
            <ResultsPage
              results={results}
              total={resultsTotal}
              currentPage={resultsPage}
              pageSize={MATCHES_PAGE_SIZE}
              onPageChange={setResultsPage}
              api={api}
            />
          )}

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
  total,
  currentPage,
  pageSize,
  onPageChange,
  busyMatchID,
  api,
  onSubmitPrediction,
}: {
  matches: MatchForUser[];
  total: number;
  currentPage: number;
  pageSize: number;
  onPageChange: (page: number) => void;
  busyMatchID: number | null;
  api: ApiClient;
  onSubmitPrediction: (matchID: number, choice: PredictionChoice) => Promise<void>;
}) {
  if (matches.length === 0) {
    return <EmptyState title="Матчей пока нет" text="Сервер еще не загрузил ближайшее расписание или сегодня нет игр." />;
  }

  return (
    <>
      <section className="grid-list">
        {matches.map((match) => (
          <MatchCard
            key={match.api_id}
            match={match}
            api={api}
            busyMatchID={busyMatchID}
            onSubmitPrediction={onSubmitPrediction}
          />
        ))}
      </section>
      <Pagination page={currentPage} total={total} pageSize={pageSize} onPageChange={onPageChange} />
    </>
  );
}

function MatchCard({
  match,
  api,
  busyMatchID,
  onSubmitPrediction,
}: {
  match: MatchForUser;
  api: ApiClient;
  busyMatchID: number | null;
  onSubmitPrediction: (matchID: number, choice: PredictionChoice) => Promise<void>;
}) {
  const [statsOpen, setStatsOpen] = useState(false);
  const [stats, setStats] = useState<PredictionStats | null>(null);
  const [statsLoading, setStatsLoading] = useState(false);
  const [statsError, setStatsError] = useState("");

  async function toggleStats() {
    if (statsOpen) {
      setStatsOpen(false);
      return;
    }

    setStatsOpen(true);
    setStatsError("");
    setStatsLoading(true);

    try {
      const data = await api.getPredictionStats(match.api_id);
      setStats(data);
    } catch (err) {
      setStatsError(err instanceof Error ? err.message : "Не удалось загрузить статистику");
    } finally {
      setStatsLoading(false);
    }
  }

  async function handlePick(choice: PredictionChoice) {
    await onSubmitPrediction(match.api_id, choice);
    if (statsOpen) {
      setStatsLoading(true);
      setStatsError("");
      try {
        const data = await api.getPredictionStats(match.api_id);
        setStats(data);
      } catch (err) {
        setStatsError(err instanceof Error ? err.message : "Не удалось обновить статистику");
      } finally {
        setStatsLoading(false);
      }
    }
  }

  return (
    <article className="match-card">
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
        onPick={handlePick}
      />

      <div className="stats-toggle-row">
        <button type="button" className="stats-toggle" onClick={() => void toggleStats()}>
          {statsOpen ? "Скрыть статистику" : "Статистика прогнозов"}
        </button>
      </div>

      {statsOpen && <PredictionStatsPanel stats={stats} loading={statsLoading} error={statsError} />}
    </article>
  );
}

function ResultsPage({
  results,
  total,
  currentPage,
  pageSize,
  onPageChange,
  api,
}: {
  results: MatchForUser[];
  total: number;
  currentPage: number;
  pageSize: number;
  onPageChange: (page: number) => void;
  api: ApiClient;
}) {
  if (results.length === 0) {
    return (
      <EmptyState
        title="Результатов пока нет"
        text="Завершенные матчи появятся здесь после загрузки результатов."
      />
    );
  }

  return (
    <>
      <section className="grid-list">
        {results.map((match) => (
          <ResultCard key={match.api_id} match={match} api={api} />
        ))}
      </section>
      <Pagination page={currentPage} total={total} pageSize={pageSize} onPageChange={onPageChange} />
    </>
  );
}

function ResultCard({ match, api }: { match: MatchForUser; api: ApiClient }) {
  const [statsOpen, setStatsOpen] = useState(false);
  const [stats, setStats] = useState<PredictionStats | null>(null);
  const [statsLoading, setStatsLoading] = useState(false);
  const [statsError, setStatsError] = useState("");
  const predictionResult = getPredictionResult(match);

  async function toggleStats() {
    if (statsOpen) {
      setStatsOpen(false);
      return;
    }

    setStatsOpen(true);
    setStatsError("");
    setStatsLoading(true);

    try {
      const data = await api.getPredictionStats(match.api_id);
      setStats(data);
    } catch (err) {
      setStatsError(err instanceof Error ? err.message : "Не удалось загрузить статистику");
    } finally {
      setStatsLoading(false);
    }
  }

  return (
    <article
      className={`match-card ${
        predictionResult === true
          ? "match-card-result-ok"
          : predictionResult === false
            ? "match-card-result-bad"
            : ""
      }`}
    >
      <div className="card-topline">
        <span>{match.league_name}</span>
        <div className="badges">
          {predictionResult !== null && (
            <span className={`result-verdict ${predictionResult ? "result-ok" : "result-bad"}`}>
              {predictionResult ? "Угадал · +1" : "Промах"}
            </span>
          )}
          <StatusBadge status={match.status} />
        </div>
      </div>

      <div className="score-row">
        <span>{match.home_team}</span>
        <strong>{formatScore(match.home_goals, match.away_goals)}</strong>
        <span>{match.away_team}</span>
      </div>

      <div className="result-footer">
        <span>
          Исход: <strong>{match.outcome ? choiceLabel(match.outcome as PredictionChoice) : "—"}</strong>
        </span>

        {match.my_prediction ? (
          <span className={predictionResult === true ? "result-ok" : predictionResult === false ? "result-bad" : ""}>
            Твой прогноз: <strong>{choiceLabel(match.my_prediction)}</strong>
          </span>
        ) : (
          <span className="result-muted">Без прогноза</span>
        )}
      </div>

      <div className="stats-toggle-row">
        <button type="button" className="stats-toggle" onClick={() => void toggleStats()}>
          {statsOpen ? "Скрыть статистику" : "Статистика прогнозов"}
        </button>
      </div>

      {statsOpen && <PredictionStatsPanel stats={stats} loading={statsLoading} error={statsError} />}
    </article>
  );
}

function Pagination({
  page,
  total,
  pageSize,
  onPageChange,
}: {
  page: number;
  total: number;
  pageSize: number;
  onPageChange: (page: number) => void;
}) {
  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  if (totalPages <= 1) {
    return null;
  }

  return (
    <nav className="pagination" aria-label="Навигация по страницам">
      <button type="button" className="pagination-btn" disabled={page <= 1} onClick={() => onPageChange(page - 1)}>
        Назад
      </button>
      <span className="pagination-info">
        Страница {page} из {totalPages}
      </span>
      <button
        type="button"
        className="pagination-btn"
        disabled={page >= totalPages}
        onClick={() => onPageChange(page + 1)}
      >
        Вперёд
      </button>
    </nav>
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

function PredictionStatsPanel({
  stats,
  loading,
  error,
}: {
  stats: PredictionStats | null;
  loading: boolean;
  error: string;
}) {
  if (loading) {
    return <div className="stats-panel stats-panel-loading">Загрузка...</div>;
  }

  if (error) {
    return <div className="stats-panel stats-panel-error">{error}</div>;
  }

  if (!stats || stats.total === 0) {
    return <div className="stats-panel stats-panel-empty">Пока никто не проголосовал</div>;
  }

  const segments = CHOICES.map((choice) => ({
    ...choice,
    percent: stats.percent[choice.value] ?? 0,
    count: stats.choices[choice.value] ?? 0,
  }));

  const votersByChoice: Record<PredictionChoice, PredictionVoter[]> = { "1": [], X: [], "2": [] };
  for (const voter of stats.voters) {
    votersByChoice[voter.user_choice].push(voter);
  }

  return (
    <div className="stats-panel">
      <div className="stats-bar" aria-label="Распределение прогнозов">
        {segments.map((segment) =>
          segment.percent > 0 ? (
            <div
              key={segment.value}
              className={`stats-bar-segment stats-bar-${segment.value.toLowerCase()}`}
              style={{ width: `${segment.percent}%` }}
              title={`${segment.label}: ${formatPercent(segment.percent)}`}
            />
          ) : null,
        )}
      </div>

      <div className="stats-legend">
        {segments.map((segment) => (
          <div key={segment.value} className="stats-legend-item">
            <span className={`stats-dot stats-dot-${segment.value.toLowerCase()}`} />
            <span>
              {segment.label}: <strong>{formatPercent(segment.percent)}</strong>
              <span className="stats-count"> ({segment.count})</span>
            </span>
          </div>
        ))}
      </div>

      <div className="stats-voters">
        {CHOICES.map((choice) => {
          const voters = votersByChoice[choice.value];
          if (voters.length === 0) {
            return null;
          }

          return (
            <div className="stats-voter-group" key={choice.value}>
              <div className="stats-voter-group-title">
                {choiceLabel(choice.value)} · {voters.length}
              </div>
              <div className="stats-voter-list">
                {voters.map((voter) => (
                  <VoterChip key={`${voter.username}-${voter.user_choice}`} voter={voter} />
                ))}
              </div>
            </div>
          );
        })}
      </div>

      <div className="stats-total">Всего прогнозов: {stats.total}</div>
    </div>
  );
}

function VoterChip({ voter }: { voter: PredictionVoter }) {
  const name = voter.display_name || voter.username || "Игрок";

  return (
    <div className="voter-chip">
      {voter.photo_url ? <img src={voter.photo_url} alt="" /> : <div className="avatar-fallback">{name[0]}</div>}
      <span>{name}</span>
    </div>
  );
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

function getPredictionResult(match: MatchForUser): boolean | null {
  if (!match.my_prediction || !match.outcome) {
    return null;
  }

  return match.my_prediction === match.outcome;
}

function formatPercent(value: number): string {
  if (value === 0) {
    return "0%";
  }
  if (Number.isInteger(value)) {
    return `${value}%`;
  }
  return `${value.toFixed(1)}%`;
}

export default App;
