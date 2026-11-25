"use client";

import Image from "next/image";
import { useRef, useState } from "react";
import type { PointerEvent } from "react";

import {
  FILES,
  PROMOTION_CHOICES,
  isOwnPiece,
  pieceIcon,
  squareAt,
  squarePosition,
} from "../../lib/chess";
import type { Color, Promotion } from "../../lib/chess";

type Props = {
  board: string[];
  side: Color;
  flipped: boolean;
  selected: number | null;
  legal: number[];
  lastFrom: number;
  lastTo: number;
  check: number;
  interactive: boolean;
  pending: { from: number; to: number } | null;
  premove: { from: number; to: number } | null;
  onSquare: (square: number) => void;
  onDragStart: (square: number) => void;
  onDrop: (from: number, to: number) => void;
  onPromote: (piece: Promotion) => void;
  onCancelPromotion: () => void;
  onCancelPremove: () => void;
};

type Drag = { from: number; face: string; x: number; y: number };

const DRAG_THRESHOLD = 5;

const FACE: Record<Promotion, string> = { q: "Q", n: "N", r: "R", b: "B" };

export default function BoardView({
  board,
  side,
  flipped,
  selected,
  legal,
  lastFrom,
  lastTo,
  check,
  interactive,
  pending,
  premove,
  onSquare,
  onDragStart,
  onDrop,
  onPromote,
  onCancelPromotion,
  onCancelPremove,
}: Props) {
  const boardRef = useRef<HTMLDivElement>(null);
  const press = useRef<{ square: number; x: number; y: number; draggable: boolean } | null>(null);

  const [drag, setDrag] = useState<Drag | null>(null);

  const locate = (event: PointerEvent) => {
    const rect = boardRef.current?.getBoundingClientRect();

    if (!rect) {
      return null;
    }

    const col = Math.floor(((event.clientX - rect.left) / rect.width) * 8);
    const row = Math.floor(((event.clientY - rect.top) / rect.height) * 8);

    if (col < 0 || col > 7 || row < 0 || row > 7) {
      return null;
    }

    return squareAt(row, col, flipped);
  };

  const onPointerDown = (event: PointerEvent) => {
    if (event.pointerType === "mouse" && event.button !== 0) {
      return;
    }

    const square = locate(event);

    if (square === null || pending) {
      return;
    }

    press.current = {
      square,
      x: event.clientX,
      y: event.clientY,
      draggable: interactive && isOwnPiece(board[square], side),
    };

    boardRef.current?.setPointerCapture(event.pointerId);
  };

  const onPointerMove = (event: PointerEvent) => {
    const start = press.current;

    if (!start) {
      return;
    }

    if (drag) {
      setDrag({ ...drag, x: event.clientX, y: event.clientY });

      return;
    }

    if (!start.draggable || Math.hypot(event.clientX - start.x, event.clientY - start.y) < DRAG_THRESHOLD) {
      return;
    }

    onDragStart(start.square);

    setDrag({ from: start.square, face: board[start.square], x: event.clientX, y: event.clientY });
  };

  const onPointerUp = (event: PointerEvent) => {
    const start = press.current;

    press.current = null;

    if (!start) {
      return;
    }

    const target = locate(event);

    if (drag) {
      setDrag(null);

      if (target !== null && target !== drag.from) {
        onDrop(drag.from, target);
      }

      return;
    }

    if (target !== null && target === start.square) {
      onSquare(target);
    }
  };

  const cancel = () => {
    press.current = null;

    setDrag(null);
  };

  const promotionSquare = pending ? squarePosition(pending.to, flipped) : null;
  const pickerAtTop = promotionSquare ? promotionSquare.row < 4 : true;

  return (
    <div
      ref={boardRef}
      className="board"
      role="grid"
      aria-label="Chess board"
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={onPointerUp}
      onPointerCancel={cancel}
      onContextMenu={(event) => {
        event.preventDefault();
        cancel();
        onCancelPremove();
      }}
    >
      {Array.from({ length: 64 }, (_, index) => {
        const row = Math.floor(index / 8);
        const col = index % 8;
        const square = squareAt(row, col, flipped);
        const face = board[square];
        const icon = pieceIcon(face);
        const dark = (row + col) % 2 === 1;
        const isLegal = legal.includes(square);
        const dragging = drag?.from === square;

        return (
          <div key={square} className={`square ${dark ? "dark" : "light"}`} role="gridcell">
            {(square === lastFrom || square === lastTo) && <span className="mark last" />}
            {square === selected && <span className="mark selected" />}
            {premove && (square === premove.from || square === premove.to) && <span className="mark premove" />}
            {square === check && <span className="mark check" />}

            {col === 0 && <span className="coord rank">{Math.floor(square / 8) + 1}</span>}
            {row === 7 && <span className="coord file">{FILES[square % 8]}</span>}

            {icon && (
              <Image
                className={`piece${dragging ? " lifted" : ""}${square === check ? " king-zoom" : ""}`}
                src={icon.src}
                alt={icon.alt}
                width={100}
                height={100}
                draggable={false}
              />
            )}

            {isLegal && <span className={icon ? "hint capture" : "hint move"} />}
          </div>
        );
      })}

      {drag && (
        <Image
          className="ghost"
          src={pieceIcon(drag.face)?.src ?? ""}
          alt=""
          width={100}
          height={100}
          draggable={false}
          style={{ left: drag.x, top: drag.y }}
        />
      )}

      {pending && promotionSquare && (
        <>
          <button
            type="button"
            className="backdrop"
            aria-label="Cancel promotion"
            onClick={onCancelPromotion}
          />

          <div
            className={`picker${pickerAtTop ? "" : " from-bottom"}`}
            style={{
              left: `${promotionSquare.col * 12.5}%`,
              top: pickerAtTop ? 0 : "50%",
            }}
          >
            {PROMOTION_CHOICES.map((choice) => {
              const icon = pieceIcon(side === "w" ? FACE[choice] : FACE[choice].toLowerCase());

              return (
                <button
                  key={choice}
                  type="button"
                  className="choice"
                  aria-label={`Promote to ${icon?.alt.replace(/^(white|black) /, "")}`}
                  onClick={() => onPromote(choice)}
                >
                  {icon && <Image src={icon.src} alt={icon.alt} width={100} height={100} draggable={false} />}
                </button>
              );
            })}
          </div>
        </>
      )}
    </div>
  );
}
