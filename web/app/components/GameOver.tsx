import { gameResult } from "../../lib/chess";
import type { Color } from "../../lib/chess";

type Props = { status: string; turn: Color; side: Color; onNewGame: () => void };

export default function GameOver({ status, turn, side, onNewGame }: Props) {
  const result = gameResult(status, turn, side);

  return (
    <div className="gameover" role="dialog" aria-label="Game over">
      <div className="card">
        <h2>{result.title}</h2>
        <p>{result.detail}</p>
        <button type="button" className="primary" onClick={onNewGame}>
          New game
        </button>
      </div>
    </div>
  );
}
