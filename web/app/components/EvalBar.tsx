import { evalFraction, evalLabel } from "../../lib/chess";
import type { EvalInfo } from "../../lib/useGameSocket";

type Props = { evaluation: EvalInfo | null; flipped: boolean };

export default function EvalBar({ evaluation, flipped }: Props) {
  const cp = evaluation?.cp ?? 0;
  const mate = evaluation?.mate ?? 0;
  const white = evalFraction(cp, mate) * 100;
  const whiteAhead = mate > 0 || (mate === 0 && cp >= 0);
  const label = evalLabel(cp, mate);

  return (
    <div
      className={`evalbar${flipped ? " flipped" : ""}`}
      role="img"
      aria-label={`Evaluation ${label}${whiteAhead ? " for White" : " for Black"}`}
    >
      <div className="black-part" style={{ height: `${100 - white}%` }} />
      <div className="white-part" style={{ height: `${white}%` }} />
      <span className={`score ${whiteAhead ? "at-white" : "at-black"}`}>{label}</span>
    </div>
  );
}
