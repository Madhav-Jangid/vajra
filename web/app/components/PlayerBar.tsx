import Image from "next/image";

import { pieceIcon } from "../../lib/chess";

type Props = {
  name: string;
  detail?: string;
  engine?: boolean;
  thinking?: boolean;
  captured: string[];
  advantage: number;
};

export default function PlayerBar({ name, detail, engine, thinking, captured, advantage }: Props) {
  return (
    <div className="player">
      <div className={`avatar${engine ? " engine" : ""}`} aria-hidden="true">
        {engine ? (
          <svg viewBox="0 0 24 24" width="22" height="22" fill="currentColor">
            <path d="M13 2 4 14h6l-1 8 9-12h-6l1-8Z" />
          </svg>
        ) : (
          <svg viewBox="0 0 24 24" width="22" height="22" fill="currentColor">
            <circle cx="12" cy="8" r="4" />
            <path d="M4 21c0-4.4 3.6-7 8-7s8 2.6 8 7H4Z" />
          </svg>
        )}
      </div>

      <div className="who">
        <div className="name">
          <span>{name}</span>
          {detail && <span className="detail">{detail}</span>}
          {thinking && (
            <span className="thinking" role="status" aria-label={`${name} is thinking`}>
              <i />
              <i />
              <i />
            </span>
          )}
        </div>

        <div className="captured" aria-label="Captured pieces">
          {captured.map((face, index) => {
            const icon = pieceIcon(face);

            return icon ? (
              <Image key={`${face}${index}`} src={icon.src} alt={icon.alt} width={40} height={40} draggable={false} />
            ) : null;
          })}

          {advantage > 0 && <span className="advantage">+{advantage}</span>}
        </div>
      </div>
    </div>
  );
}
