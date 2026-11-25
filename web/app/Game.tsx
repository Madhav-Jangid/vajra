"use client";

import { useState } from "react";

import { capturedPieces, isOwnPiece, isPromotionMove, premoveTargets } from "../lib/chess";
import type { Promotion } from "../lib/chess";
import { useGameSocket } from "../lib/useGameSocket";
import type { SideChoice } from "../lib/useGameSocket";
import BoardView from "./components/BoardView";
import EvalBar from "./components/EvalBar";
import GameOver from "./components/GameOver";
import MoveList from "./components/MoveList";
import PlayerBar from "./components/PlayerBar";

type Selection = { square: number; version: number };

type Pending = { from: number; to: number; version: number };

const SIDES: { value: SideChoice; label: string }[] = [
  { value: "w", label: "White" },
  { value: "random", label: "Random" },
  { value: "b", label: "Black" },
];

export default function Game() {
  const { state, connected, legal, premove, requestMoves, sendMove, setPremove, reset, undo, clearLegal } =
    useGameSocket();

  const [manualFlip, setManualFlip] = useState(false);
  const [choice, setChoice] = useState<SideChoice | null>(null);
  const [selection, setSelection] = useState<Selection | null>(null);
  const [promotion, setPromotion] = useState<Pending | null>(null);

  const side = state.side;
  const flipped = (side === "b") !== manualFlip;
  const picked = choice ?? side;

  const thinking = state.turn !== side && !state.over;
  const myTurn = connected && state.turn === side && !state.over;
  const canPremove = connected && thinking;
  const canAct = myTurn || canPremove;

  const selected = selection && selection.version === state.version ? selection.square : null;
  const pending = promotion && promotion.version === state.version ? promotion : null;

  const targets = (square: number | null) => {
    if (square === null) {
      return [];
    }

    if (canPremove) {
      return premoveTargets(state.board, square);
    }

    return legal && legal.square === square ? legal.moves : [];
  };

  const legalMoves = targets(selected);

  const captured = capturedPieces(state.board);
  const mine = side === "w" ? captured.byWhite : captured.byBlack;
  const theirs = side === "w" ? captured.byBlack : captured.byWhite;
  const lead = side === "w" ? captured.whiteAdvantage : -captured.whiteAdvantage;

  const deselect = () => {
    setSelection(null);
    clearLegal();
  };

  const select = (square: number) => {
    setSelection({ square, version: state.version });

    if (myTurn) {
      requestMoves(square);
    }
  };

  const play = (from: number, to: number, piece?: Promotion) => {
    if (canPremove) {
      setPremove({ from, to, promotion: piece });

      return;
    }

    sendMove(from, to, piece);
  };

  const commit = (from: number, to: number) => {
    deselect();

    if (isPromotionMove(state.board, from, to)) {
      setPromotion({ from, to, version: state.version });

      return;
    }

    play(from, to);
  };

  const onSquare = (square: number) => {
    if (!canAct) {
      return;
    }

    if (selected !== null && legalMoves.includes(square)) {
      commit(selected, square);

      return;
    }

    if (isOwnPiece(state.board[square], side) && square !== selected) {
      select(square);

      return;
    }

    deselect();
  };

  const onDrop = (from: number, to: number) => {
    if (!canAct) {
      return;
    }

    const known = canPremove ? premoveTargets(state.board, from) : legal && legal.square === from ? legal.moves : null;

    if (known && !known.includes(to)) {
      return;
    }

    commit(from, to);
  };

  const onPromote = (piece: Promotion) => {
    if (pending) {
      play(pending.from, pending.to, piece);
    }

    setPromotion(null);
  };

  const startGame = (as: SideChoice) => {
    deselect();
    setPromotion(null);
    setManualFlip(false);
    reset(as);
  };

  const chooseSide = (value: SideChoice) => {
    setChoice(value);

    if (state.history.length === 0 && connected) {
      startGame(value);
    }
  };

  const cancelPremove = () => {
    setPremove(null);
    deselect();
  };

  const message = !connected
    ? "Connecting to the game server…"
    : state.over
      ? "Game over"
      : thinking
        ? premove
          ? "Vajra is thinking… premove set"
          : "Vajra is thinking…"
        : state.hasCheck
          ? "Check! Your move"
          : "Your move";

  const vajra = (
    <PlayerBar
      name="Vajra"
      detail={state.eval ? `depth ${state.eval.depth}` : "engine"}
      engine
      thinking={thinking}
      captured={theirs}
      advantage={Math.max(-lead, 0)}
    />
  );

  const you = <PlayerBar name="You" captured={mine} advantage={Math.max(lead, 0)} />;

  const canUndo = connected && !thinking && state.history.length > (side === "b" ? 1 : 0);

  return (
    <main className="app">
      <header className="topbar">
        <svg viewBox="0 0 24 24" width="26" height="26" fill="currentColor" aria-hidden="true">
          <path d="M13 2 4 14h6l-1 8 9-12h-6l1-8Z" />
        </svg>
        <span>Vajra</span>
      </header>

      <div className="layout">
        <section className="play">
          <EvalBar evaluation={state.eval} flipped={flipped} />

          <div className="column">
            {flipped ? you : vajra}

            <div className="board-frame">
              <BoardView
                board={state.board}
                side={side}
                flipped={flipped}
                selected={selected}
                legal={legalMoves}
                lastFrom={state.lastFrom}
                lastTo={state.lastTo}
                check={state.hasCheck ? state.check : -1}
                interactive={canAct}
                pending={pending}
                premove={premove}
                onSquare={onSquare}
                onDragStart={select}
                onDrop={onDrop}
                onPromote={onPromote}
                onCancelPromotion={() => setPromotion(null)}
                onCancelPremove={cancelPremove}
              />

              {state.over && (
                <GameOver status={state.status} turn={state.turn} side={side} onNewGame={() => startGame(picked)} />
              )}
            </div>

            {flipped ? vajra : you}
          </div>
        </section>

        <aside className="panel">
          <h1>Play Vajra</h1>

          <p className={`status${connected ? "" : " offline"}`} role="status">
            {message}
          </p>

          {!connected && (
            <p className="hint-text">
              Start the server with <code>go run ./cmd/server</code>.
            </p>
          )}

          {thinking && !premove && <p className="hint-text">Drag a piece now to premove. Right-click cancels.</p>}

          <MoveList history={state.history} />

          <div className="sides" role="group" aria-label="Play as">
            <span>Play as</span>

            {SIDES.map((option) => (
              <button
                key={option.value}
                type="button"
                aria-pressed={picked === option.value}
                disabled={!connected}
                onClick={() => chooseSide(option.value)}
              >
                {option.label}
              </button>
            ))}
          </div>

          <div className="controls">
            <button type="button" className="primary" onClick={() => startGame(picked)} disabled={!connected}>
              New game
            </button>
            <button type="button" onClick={undo} disabled={!canUndo}>
              Undo
            </button>
            <button type="button" onClick={() => setManualFlip(!manualFlip)}>
              Flip board
            </button>
          </div>
        </aside>
      </div>
    </main>
  );
}
