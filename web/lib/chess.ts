export type Color = "w" | "b";

export const FILES = "abcdefgh";

const KINDS: Record<string, string> = {
  k: "king",
  q: "queen",
  r: "rook",
  n: "knight",
  b: "bishop",
  p: "pawn",
};

const START_COUNTS: Record<string, number> = { q: 1, r: 2, b: 2, n: 2, p: 8 };

const VALUES: Record<string, number> = { p: 1, n: 3, b: 3, r: 5, q: 9 };

const CAPTURE_ORDER = ["q", "r", "b", "n", "p"];

export const PROMOTION_CHOICES = ["q", "n", "r", "b"] as const;

export type Promotion = (typeof PROMOTION_CHOICES)[number];

export function squareName(square: number): string {
  return FILES[square % 8] + (Math.floor(square / 8) + 1);
}

export function isWhitePiece(face: string): boolean {
  return face !== "." && face === face.toUpperCase();
}

export function isOwnPiece(face: string, side: Color): boolean {
  return face !== "." && isWhitePiece(face) === (side === "w");
}

export function pieceIcon(face: string) {
  const kind = KINDS[face.toLowerCase()];

  if (!kind) {
    return null;
  }

  const color = isWhitePiece(face) ? "w" : "b";

  return {
    src: `/icons/png/${kind}-${color}.png`,
    alt: `${color === "w" ? "white" : "black"} ${kind}`,
  };
}

export function squareAt(row: number, col: number, flipped: boolean): number {
  return flipped ? row * 8 + (7 - col) : (7 - row) * 8 + col;
}

export function squarePosition(square: number, flipped: boolean) {
  const rank = Math.floor(square / 8);
  const file = square % 8;

  return flipped
    ? { row: rank, col: 7 - file }
    : { row: 7 - rank, col: file };
}

export function isPromotionMove(
  board: string[],
  from: number,
  to: number,
): boolean {
  const face = board[from];

  if (face === "P") {
    return Math.floor(to / 8) === 7;
  }

  if (face === "p") {
    return Math.floor(to / 8) === 0;
  }

  return false;
}

export function capturedPieces(board: string[]) {
  const onBoard: Record<string, number> = {};

  for (const face of board) {
    if (face !== ".") {
      onBoard[face] = (onBoard[face] ?? 0) + 1;
    }
  }

  const missing = (upper: boolean) => {
    const faces: string[] = [];
    let value = 0;

    for (const kind of CAPTURE_ORDER) {
      const face = upper ? kind.toUpperCase() : kind;
      const gone = Math.max(START_COUNTS[kind] - (onBoard[face] ?? 0), 0);

      for (let i = 0; i < gone; i++) {
        faces.push(face);
      }

      value += gone * VALUES[kind];
    }

    return { faces, value };
  };

  const takenFromBlack = missing(false);
  const takenFromWhite = missing(true);

  return {
    byWhite: takenFromBlack.faces,
    byBlack: takenFromWhite.faces,
    whiteAdvantage: takenFromBlack.value - takenFromWhite.value,
  };
}

export function movePairs(history: string[]) {
  const pairs: { number: number; white: string; black: string | null }[] = [];

  for (let i = 0; i < history.length; i += 2) {
    pairs.push({
      number: i / 2 + 1,
      white: history[i],
      black: history[i + 1] ?? null,
    });
  }

  return pairs;
}

export function evalFraction(cp: number, mate: number): number {
  if (mate > 0) {
    return 1;
  }

  if (mate < 0) {
    return 0;
  }

  return 1 / (1 + Math.exp(-cp / 400));
}

export function evalLabel(cp: number, mate: number): string {
  if (mate !== 0) {
    return `M${Math.abs(mate)}`;
  }

  const pawns = cp / 100;

  if (Math.abs(pawns) < 0.05) {
    return "0.0";
  }

  return `${pawns > 0 ? "+" : ""}${pawns.toFixed(1)}`;
}

export function gameResult(status: string, turn: Color, side: Color = "w") {
  if (status === "checkmate") {
    const youWon = turn !== side;

    return {
      title: youWon ? "You won" : "Vajra won",
      detail: "by checkmate",
      youWon,
      draw: false,
    };
  }

  const reasons: Record<string, string> = {
    stalemate: "by stalemate",
    "insufficient material": "by insufficient material",
    "fifty-move rule": "by the fifty-move rule",
    "threefold repetition": "by repetition",
  };

  return {
    title: "Draw",
    detail: reasons[status] ?? status,
    youWon: false,
    draw: true,
  };
}

const KNIGHT_JUMPS = [
  [1, 2],
  [2, 1],
  [2, -1],
  [1, -2],
  [-1, -2],
  [-2, -1],
  [-2, 1],
  [-1, 2],
];

const ORTHOGONAL = [
  [1, 0],
  [-1, 0],
  [0, 1],
  [0, -1],
];

const DIAGONAL = [
  [1, 1],
  [1, -1],
  [-1, 1],
  [-1, -1],
];

export function premoveTargets(board: string[], from: number): number[] {
  const face = board[from];

  if (!face || face === ".") {
    return [];
  }

  const white = isWhitePiece(face);
  const rank = Math.floor(from / 8);
  const file = from % 8;
  const targets: number[] = [];

  const inside = (r: number, f: number) => r >= 0 && r < 8 && f >= 0 && f < 8;

  const reach = (r: number, f: number) => {
    if (!inside(r, f)) {
      return false;
    }

    const target = board[r * 8 + f];

    if (target !== "." && isWhitePiece(target) === white) {
      return false;
    }

    targets.push(r * 8 + f);

    return target === ".";
  };

  const slide = (directions: number[][]) => {
    for (const [dr, df] of directions) {
      let r = rank + dr;
      let f = file + df;

      while (reach(r, f)) {
        r += dr;
        f += df;
      }
    }
  };

  switch (face.toLowerCase()) {
    case "n":
      for (const [dr, df] of KNIGHT_JUMPS) {
        reach(rank + dr, file + df);
      }

      break;
    case "k":
      for (const [dr, df] of [...ORTHOGONAL, ...DIAGONAL]) {
        reach(rank + dr, file + df);
      }

      break;
    case "b":
      slide(DIAGONAL);

      break;
    case "r":
      slide(ORTHOGONAL);

      break;
    case "q":
      slide([...ORTHOGONAL, ...DIAGONAL]);

      break;
    case "p": {
      const step = white ? 1 : -1;
      const ahead = rank + step;

      if (inside(ahead, file) && board[ahead * 8 + file] === ".") {
        targets.push(ahead * 8 + file);

        const home = white ? 1 : 6;
        const two = rank + 2 * step;

        if (rank === home && board[two * 8 + file] === ".") {
          targets.push(two * 8 + file);
        }
      }

      for (const df of [-1, 1]) {
        if (!inside(ahead, file + df)) {
          continue;
        }

        const target = board[ahead * 8 + file + df];

        if (target !== "." && isWhitePiece(target) !== white) {
          targets.push(ahead * 8 + file + df);
        }
      }

      break;
    }
  }

  return targets;
}
