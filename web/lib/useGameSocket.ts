"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import confetti from "canvas-confetti";

import type { Color, Promotion } from "./chess";
import { playSound } from "./sounds";

export type EvalInfo = { cp: number; mate: number; depth: number };

export type GameState = {
  board: string[];
  turn: Color;
  side: Color;
  history: string[];
  lastFrom: number;
  lastTo: number;
  hasCheck: boolean;
  check: number;
  over: boolean;
  status: string;
  eval: EvalInfo | null;
  version: number;
};

export type SideChoice = "w" | "b" | "random";

export type LegalMoves = { square: number; moves: number[] };

export type Premove = { from: number; to: number; promotion?: Promotion };

type ServerMessage = {
  type: "state" | "moves";
  board?: string[];
  turn?: Color;
  side?: Color;
  event?: string;
  history?: string[];
  lastFrom?: number;
  lastTo?: number;
  hasCheck?: boolean;
  check?: number;
  over?: boolean;
  status?: string;
  eval?: EvalInfo;
  square?: number;
  moves?: number[];
};

const INITIAL: GameState = {
  board: Array(64).fill("."),
  turn: "w",
  side: "w",
  history: [],
  lastFrom: -1,
  lastTo: -1,
  hasCheck: false,
  check: -1,
  over: false,
  status: "",
  eval: null,
  version: 0,
};

const SERVER_PORT = 8080;

function celebrate() {
  confetti({ particleCount: 180, spread: 100, origin: { y: 0.6 } });

  setTimeout(() => {
    confetti({ particleCount: 100, spread: 120, origin: { x: 0.2, y: 0.6 } });
    confetti({ particleCount: 100, spread: 120, origin: { x: 0.8, y: 0.6 } });
  }, 250);
}

export function useGameSocket() {
  const [state, setState] = useState<GameState>(INITIAL);
  const [connected, setConnected] = useState(false);
  const [legal, setLegal] = useState<LegalMoves | null>(null);
  const [premove, setPremoveState] = useState<Premove | null>(null);

  const socket = useRef<WebSocket | null>(null);
  const queued = useRef<Premove | null>(null);
  const gameEndPlayed = useRef(false);

  useEffect(() => {
    let closed = false;
    let reconnect: ReturnType<typeof setTimeout>;

    const handle = (message: ServerMessage) => {
      if (message.type === "moves") {
        setLegal({ square: message.square ?? -1, moves: message.moves ?? [] });

        return;
      }

      if (message.type !== "state" || !message.board) {
        return;
      }

      const turn = message.turn ?? "w";
      const side = message.side ?? "w";
      const over = message.over === true;
      const checkmate = message.status === "checkmate";

      setLegal(null);

      setState((previous) => ({
        board: message.board ?? previous.board,
        turn,
        side,
        history: message.history ?? [],
        lastFrom: message.lastFrom ?? -1,
        lastTo: message.lastTo ?? -1,
        hasCheck: message.hasCheck === true,
        check: message.check ?? -1,
        over,
        status: message.status ?? "",
        eval: message.eval ?? null,
        version: previous.version + 1,
      }));

      if (message.event && !checkmate) {
        playSound(message.event);
      }

      if (checkmate && !gameEndPlayed.current) {
        gameEndPlayed.current = true;

        playSound("game-end");

        if (turn !== side) {
          celebrate();
        }
      }

      if (!over) {
        gameEndPlayed.current = false;
      }

      const next = queued.current;

      if (next && (over || turn === side)) {
        queued.current = null;

        setPremoveState(null);

        if (!over) {
          socket.current?.send(JSON.stringify({ type: "move", ...next }));
        }
      }
    };

    const connect = () => {
      const ws = new WebSocket(`ws://${window.location.hostname}:${SERVER_PORT}/ws`);

      socket.current = ws;

      ws.onopen = () => {
        setConnected(true);

        ws.send(JSON.stringify({ type: "state" }));
      };

      ws.onclose = () => {
        setConnected(false);

        if (!closed) {
          reconnect = setTimeout(connect, 1000);
        }
      };

      ws.onmessage = (event) => handle(JSON.parse(event.data));
    };

    connect();

    return () => {
      closed = true;

      clearTimeout(reconnect);

      socket.current?.close();
    };
  }, []);

  const send = useCallback((message: object) => {
    if (socket.current?.readyState === WebSocket.OPEN) {
      socket.current.send(JSON.stringify(message));
    }
  }, []);

  const requestMoves = useCallback(
    (square: number) => send({ type: "moves", square }),
    [send],
  );

  const sendMove = useCallback(
    (from: number, to: number, promotion?: Promotion) =>
      send({ type: "move", from, to, promotion }),
    [send],
  );

  const setPremove = useCallback((move: Premove | null) => {
    queued.current = move;

    setPremoveState(move);
  }, []);

  const reset = useCallback((side?: SideChoice) => {
    setPremove(null);

    send({ type: "reset", side });
  }, [send, setPremove]);

  const undo = useCallback(() => {
    setPremove(null);

    send({ type: "undo" });
  }, [send, setPremove]);

  const clearLegal = useCallback(() => setLegal(null), []);

  return {
    state,
    connected,
    legal,
    premove,
    requestMoves,
    sendMove,
    setPremove,
    reset,
    undo,
    clearLegal,
  };
}
