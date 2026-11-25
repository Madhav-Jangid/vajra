"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import confetti from "canvas-confetti";
import Image from "next/image";

const WS_URL = "ws://localhost:8080/ws";

const KINDS: Record<string, string> = {
  k: "king",
  q: "queen",
  r: "rook",
  n: "knight",
  b: "bishop",
  p: "pawn",
};

function iconFor(face: string) {
  const kind = KINDS[face.toLowerCase()];

  if (!kind) {
    return null;
  }

  const color = face === face.toUpperCase() ? "w" : "b";

  return {
    src: `/icons/png/${kind}-${color}.png`,
    alt: `${color === "w" ? "white" : "black"} ${kind}`,
  };
}

type ServerMsg = {
  type: "state" | "moves";

  board?: string[];

  turn?: "w" | "b";

  square?: number;
  moves?: number[];

  event?: string;

  hasCheck?: boolean;
  check?: number;

  over?: boolean;
  status?: string;
};

export default function Board() {
  const [board, setBoard] = useState<string[]>(
    Array(64).fill("."),
  );

  const [turn, setTurn] = useState<"w" | "b">("w");

  const [selected, setSelected] = useState<number | null>(
    null,
  );

  const [moves, setMoves] = useState<number[]>([]);

  const [lastMove, setLastMove] = useState<number[]>([]);

  const [online, setOnline] = useState(false);

  // -1 means no checked king.
  const [checkSq, setCheckSq] = useState<number>(-1);

  const [result, setResult] = useState("");

  const ws = useRef<WebSocket | null>(null);

  const audio = useRef<Record<string, HTMLAudioElement>>({});

  const previousBoard = useRef<string[] | null>(null);

  const gameEndPlayed = useRef(false);

  const play = useCallback((name: string) => {
    try {
      const audioElement =
        audio.current[name] ??
        (audio.current[name] = new Audio(
          `/sounds/${name}.mp3`,
        ));

      audioElement.currentTime = 0;

      void audioElement.play().catch(() => {});
    } catch {
      // Ignore browser audio restrictions.
    }
  }, []);

  const celebrate = useCallback(() => {
    confetti({
      particleCount: 180,
      spread: 100,
      origin: {
        y: 0.6,
      },
    });

    setTimeout(() => {
      confetti({
        particleCount: 100,
        spread: 120,
        origin: {
          x: 0.2,
          y: 0.6,
        },
      });

      confetti({
        particleCount: 100,
        spread: 120,
        origin: {
          x: 0.8,
          y: 0.6,
        },
      });
    }, 250);
  }, []);

  useEffect(() => {
    let closed = false;

    let reconnectTimer: ReturnType<
      typeof setTimeout
    >;

    const connect = () => {
      const socket = new WebSocket(WS_URL);

      ws.current = socket;

      socket.onopen = () => {
        setOnline(true);

        // Ask the server for the current state.
        socket.send(
          JSON.stringify({
            type: "state",
          }),
        );
      };

      socket.onclose = () => {
        setOnline(false);

        if (!closed) {
          reconnectTimer = setTimeout(connect, 1000);
        }
      };

      socket.onmessage = (event) => {
        const message: ServerMsg =
          JSON.parse(event.data);

        // ----------------------------------------
        // Legal moves response
        // ----------------------------------------

        if (message.type === "moves") {
          setMoves(message.moves ?? []);
          return;
        }

        // ----------------------------------------
        // Board state response
        // ----------------------------------------

        if (
          message.type !== "state" ||
          !message.board
        ) {
          return;
        }

        const oldBoard = previousBoard.current;

        if (oldBoard) {
          const changed = message.board
            .map((piece, index) =>
              piece !== oldBoard[index]
                ? index
                : -1,
            )
            .filter(
              (index) => index !== -1,
            );

          setLastMove(changed);
        }

        previousBoard.current =
          message.board;

        setBoard(message.board);

        setTurn(
          message.turn ?? "w",
        );

        setSelected(null);
        setMoves([]);

        const isGameOver =
          message.over === true;

        const isCheckmate =
          message.status ===
          "checkmate";

        setResult(
          isGameOver
            ? message.status ??
                "game over"
            : "",
        );

        // ----------------------------------------
        // Check highlighting
        // ----------------------------------------

        if (
          message.hasCheck &&
          message.check !== undefined &&
          message.check >= 0
        ) {
          setCheckSq(
            message.check,
          );
        } else if (
          !isGameOver
        ) {
          setCheckSq(-1);
        }

        // ----------------------------------------
        // Move sounds
        // ----------------------------------------

        if (
          message.event &&
          !isCheckmate
        ) {
          play(message.event);
        }

        // ----------------------------------------
        // Checkmate / game end
        // ----------------------------------------

        if (
          isCheckmate &&
          !gameEndPlayed.current
        ) {
          gameEndPlayed.current =
            true;

          play("game-end");

          celebrate();
        }

        // ----------------------------------------
        // Reset game-end guard
        // ----------------------------------------

        if (!isGameOver) {
          gameEndPlayed.current =
            false;
        }
      };
    };

    connect();

    return () => {
      closed = true;

      clearTimeout(
        reconnectTimer,
      );

      ws.current?.close();
    };
  }, [play, celebrate]);

  const send = useCallback(
    (message: object) => {
      if (
        ws.current?.readyState ===
        WebSocket.OPEN
      ) {
        ws.current.send(
          JSON.stringify(message),
        );
      }
    },
    [],
  );

  const click = (square: number) => {
    // ----------------------------------------
    // Move selected piece
    // ----------------------------------------

    if (
      selected !== null &&
      moves.includes(square)
    ) {
      send({
        type: "move",
        from: selected,
        to: square,
      });

      return;
    }

    const piece = board[square];

    const isMine =
      piece !== "." &&
      (piece === piece.toUpperCase()
        ? "w"
        : "b") === turn;

    // ----------------------------------------
    // Select own piece
    // ----------------------------------------

    if (isMine) {
      setSelected(square);

      setMoves([]);

      send({
        type: "moves",
        square,
      });

      return;
    }

    // ----------------------------------------
    // Clear selection
    // ----------------------------------------

    setSelected(null);

    setMoves([]);
  };

  const winner =
    result === "checkmate"
      ? turn === "w"
        ? "Black"
        : "White"
      : null;

  return (
    <div className="flex flex-col items-center gap-3">
      <div
        className="grid grid-cols-8 grid-rows-8 overflow-hidden rounded"
        style={{
          width: "var(--size)",
          height: "var(--size)",
        }}
      >
        {Array.from(
          { length: 64 },
          (_, i) => {
            const row =
              Math.floor(i / 8);

            const col =
              i % 8;

            // UI is rendered top-to-bottom,
            // chess board is a1 = 0.
            const square =
              (7 - row) * 8 +
              col;

            const dark =
              (row + col) % 2 === 1;

            const selectedSquare =
              square === selected;

            const changedSquare =
              lastMove.includes(
                square,
              );

            const legalMove =
              moves.includes(
                square,
              );

            const checkedKing =
              square === checkSq;

            const piece =
              iconFor(
                board[square],
              );

            return (
              <button
                key={square}
                onClick={() =>
                  click(square)
                }
                className={`relative flex items-center justify-center p-0 ${
                  checkedKing
                    ? "check-flash"
                    : ""
                }`}
                style={{
                  background:
                    selectedSquare ||
                    changedSquare
                      ? "#bbcb2b"
                      : dark
                        ? "#769656"
                        : "#eeeed2",
                }}
              >
                {piece && (
                  <Image
                    width={100}
                    height={100}
                    src={piece.src}
                    alt={piece.alt}
                    draggable={false}
                    className={`relative h-[90%] w-[90%] select-none ${
                      checkedKing
                        ? "king-zoom"
                        : ""
                    }`}
                  />
                )}

                {legalMove && (
                  <span
                    className={
                      piece
                        ? "pointer-events-none absolute inset-[4%] rounded-full border-[5px] border-black/25"
                        : "pointer-events-none absolute h-[30%] w-[30%] rounded-full bg-black/25"
                    }
                  />
                )}
              </button>
            );
          },
        )}
      </div>

      <div
        className="text-sm"
        style={{
          color:
            "var(--muted)",
        }}
      >
        {!online ? (
          "Connecting to server…"
        ) : result ===
          "checkmate" ? (
          `Checkmate – ${winner} wins`
        ) : result ? (
          `Draw – ${result}`
        ) : (
          `${
            turn === "w"
              ? "White"
              : "Black"
          } to move`
        )}
      </div>

      <button
        onClick={() => {
          gameEndPlayed.current =
            false;

          setCheckSq(-1);
          setResult("");

          send({
            type: "reset",
          });
        }}
        className="rounded px-3 py-1 text-sm"
        style={{
          background:
            "var(--green)",
          color: "#fff",
        }}
      >
        New game
      </button>
    </div>
  );
}