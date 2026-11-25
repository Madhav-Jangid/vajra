"use client";

import { useEffect, useRef } from "react";

import { movePairs } from "../../lib/chess";

export default function MoveList({ history }: { history: string[] }) {
  const scroller = useRef<HTMLDivElement>(null);

  useEffect(() => {
    scroller.current?.scrollTo({ top: scroller.current.scrollHeight });
  }, [history.length]);

  const pairs = movePairs(history);
  const last = history.length - 1;

  return (
    <div className="moves" ref={scroller} aria-label="Moves">
      {pairs.length === 0 && <p className="empty">Make a move to start the game.</p>}

      {pairs.map((pair, index) => (
        <div className="pair" key={pair.number}>
          <span className="number">{pair.number}.</span>
          <span className={`ply${index * 2 === last ? " current" : ""}`}>{pair.white}</span>
          <span className={`ply${index * 2 + 1 === last ? " current" : ""}`}>{pair.black ?? ""}</span>
        </div>
      ))}
    </div>
  );
}
