import assert from "node:assert/strict";
import { test } from "node:test";

import {
  capturedPieces,
  evalFraction,
  evalLabel,
  gameResult,
  isOwnPiece,
  isPromotionMove,
  isWhitePiece,
  movePairs,
  pieceIcon,
  premoveTargets,
  squareAt,
  squareName,
  squarePosition,
} from "./chess.ts";

const START = [
  ..."RNBQKBNR",
  ..."PPPPPPPP",
  ...".".repeat(32),
  ..."pppppppp",
  ..."rnbqkbnr",
];

test("square names run from a1 to h8", () => {
  assert.equal(squareName(0), "a1");
  assert.equal(squareName(7), "h1");
  assert.equal(squareName(28), "e4");
  assert.equal(squareName(63), "h8");
});

test("white pieces are upper case", () => {
  assert.equal(isWhitePiece("K"), true);
  assert.equal(isWhitePiece("k"), false);
  assert.equal(isWhitePiece("."), false);
});

test("piece icons follow colour and kind", () => {
  assert.deepEqual(pieceIcon("N"), {
    src: "/icons/png/knight-w.png",
    alt: "white knight",
  });
  assert.deepEqual(pieceIcon("q"), {
    src: "/icons/png/queen-b.png",
    alt: "black queen",
  });
  assert.equal(pieceIcon("."), null);
});

test("board orientation maps screen squares to board squares and back", () => {
  assert.equal(squareAt(7, 0, false), 0);
  assert.equal(squareAt(0, 7, false), 63);
  assert.equal(squareAt(0, 0, true), 7);
  assert.equal(squareAt(7, 7, true), 56);

  for (let square = 0; square < 64; square++) {
    for (const flipped of [false, true]) {
      const { row, col } = squarePosition(square, flipped);

      assert.equal(squareAt(row, col, flipped), square);
    }
  }

  assert.deepEqual(squarePosition(0, false), { row: 7, col: 0 });
  assert.deepEqual(squarePosition(0, true), { row: 0, col: 7 });
});

test("only a pawn reaching the last rank is a promotion", () => {
  const board = [...".".repeat(64)];

  board[48] = "P";
  board[8] = "p";
  board[12] = "P";
  board[0] = "R";

  assert.equal(isPromotionMove(board, 48, 56), true);
  assert.equal(isPromotionMove(board, 48, 57), true);
  assert.equal(isPromotionMove(board, 12, 20), false);
  assert.equal(isPromotionMove(board, 8, 0), true);
  assert.equal(isPromotionMove(board, 8, 16), false);
  assert.equal(isPromotionMove(board, 0, 56), false);
  assert.equal(isPromotionMove(board, 30, 38), false);
});

test("nothing is captured at the start", () => {
  assert.deepEqual(capturedPieces(START), {
    byWhite: [],
    byBlack: [],
    whiteAdvantage: 0,
  });
});

test("captured pieces and material difference", () => {
  const board = [...START];

  board[59] = ".";
  board[57] = ".";
  board[11] = ".";
  board[12] = ".";

  const result = capturedPieces(board);

  assert.deepEqual(result.byWhite, ["q", "n"]);
  assert.deepEqual(result.byBlack, ["P", "P"]);
  assert.equal(result.whiteAdvantage, 9 + 3 - 2);
});

test("pieces are listed from most to least valuable", () => {
  const board = [...START];

  board[48] = ".";
  board[56] = ".";
  board[58] = ".";

  assert.deepEqual(capturedPieces(board).byWhite, ["r", "b", "p"]);
});

test("a promoted piece does not make a negative capture count", () => {
  const board = [...START];

  board[48] = ".";
  board[56] = "Q";
  board[3] = "Q";

  const result = capturedPieces(board);

  assert.deepEqual(result.byBlack, []);
  assert.deepEqual(result.byWhite, ["r", "p"]);
});

test("moves are paired by number", () => {
  assert.deepEqual(movePairs([]), []);
  assert.deepEqual(movePairs(["e4"]), [{ number: 1, white: "e4", black: null }]);
  assert.deepEqual(movePairs(["e4", "e5", "Nf3"]), [
    { number: 1, white: "e4", black: "e5" },
    { number: 2, white: "Nf3", black: null },
  ]);
});

test("the evaluation bar fills towards the side that is better", () => {
  assert.equal(evalFraction(0, 0), 0.5);
  assert.ok(evalFraction(300, 0) > 0.6);
  assert.ok(evalFraction(-300, 0) < 0.4);
  assert.ok(evalFraction(2000, 0) < 1);
  assert.ok(evalFraction(-2000, 0) > 0);
  assert.equal(evalFraction(0, 3), 1);
  assert.equal(evalFraction(0, -2), 0);
  assert.ok(Math.abs(evalFraction(150, 0) + evalFraction(-150, 0) - 1) < 1e-12);
});

test("evaluation labels", () => {
  assert.equal(evalLabel(0, 0), "0.0");
  assert.equal(evalLabel(3, 0), "0.0");
  assert.equal(evalLabel(36, 0), "+0.4");
  assert.equal(evalLabel(-120, 0), "-1.2");
  assert.equal(evalLabel(0, 3), "M3");
  assert.equal(evalLabel(0, -2), "M2");
});

test("results say who won and why", () => {
  assert.deepEqual(gameResult("checkmate", "b"), {
    title: "You won",
    detail: "by checkmate",
    youWon: true,
    draw: false,
  });
  assert.deepEqual(gameResult("checkmate", "w"), {
    title: "Vajra won",
    detail: "by checkmate",
    youWon: false,
    draw: false,
  });

  for (const [status, detail] of [
    ["stalemate", "by stalemate"],
    ["insufficient material", "by insufficient material"],
    ["fifty-move rule", "by the fifty-move rule"],
    ["threefold repetition", "by repetition"],
  ]) {
    const result = gameResult(status, "w");

    assert.equal(result.title, "Draw");
    assert.equal(result.detail, detail);
    assert.equal(result.draw, true);
  }

  assert.equal(gameResult("something new", "w").detail, "something new");
});

function boardWith(pieces: Record<string, string>) {
  const board = [...".".repeat(64)];

  for (const [name, face] of Object.entries(pieces)) {
    board["abcdefgh".indexOf(name[0]) + (Number(name[1]) - 1) * 8] = face;
  }

  return board;
}

function targetNames(board: string[], from: string) {
  const start = "abcdefgh".indexOf(from[0]) + (Number(from[1]) - 1) * 8;

  return premoveTargets(board, start).map(squareName).sort();
}

test("premove targets: an empty square or a missing piece has none", () => {
  assert.deepEqual(premoveTargets(boardWith({}), 20), []);
  assert.deepEqual(premoveTargets([], 5), []);
});

test("premove targets: knight", () => {
  assert.deepEqual(targetNames(boardWith({ d4: "N" }), "d4"), [
    "b3", "b5", "c2", "c6", "e2", "e6", "f3", "f5",
  ]);
  assert.deepEqual(targetNames(boardWith({ a1: "N" }), "a1"), ["b3", "c2"]);
  assert.deepEqual(targetNames(boardWith({ a1: "N", b3: "P", c2: "p" }), "a1"), ["c2"]);
});

test("premove targets: king", () => {
  assert.equal(targetNames(boardWith({ e4: "K" }), "e4").length, 8);
  assert.deepEqual(targetNames(boardWith({ h1: "K" }), "h1"), ["g1", "g2", "h2"]);
});

test("premove targets: sliders stop at blockers and take enemies", () => {
  const rook = boardWith({ d4: "R", d6: "P", d2: "p", g4: "N" });

  assert.deepEqual(targetNames(rook, "d4"), [
    "a4", "b4", "c4", "d2", "d3", "d5", "e4", "f4",
  ]);

  const bishop = boardWith({ c1: "B", e3: "p", b2: "P" });

  assert.deepEqual(targetNames(bishop, "c1"), ["d2", "e3"]);

  assert.equal(targetNames(boardWith({ d4: "Q" }), "d4").length, 27);
});

test("premove targets: pawn pushes", () => {
  assert.deepEqual(targetNames(boardWith({ e2: "P" }), "e2"), ["e3", "e4"]);
  assert.deepEqual(targetNames(boardWith({ e3: "P" }), "e3"), ["e4"]);
  assert.deepEqual(targetNames(boardWith({ e2: "P", e3: "p" }), "e2"), []);
  assert.deepEqual(targetNames(boardWith({ e2: "P", e4: "p" }), "e2"), ["e3"]);
  assert.deepEqual(targetNames(boardWith({ e7: "p" }), "e7"), ["e5", "e6"]);
});

test("premove targets: pawn captures only enemy pieces", () => {
  assert.deepEqual(targetNames(boardWith({ e4: "P", d5: "p", f5: "P" }), "e4"), ["d5", "e5"]);
  assert.deepEqual(targetNames(boardWith({ a4: "P", b5: "p" }), "a4"), ["a5", "b5"]);
  assert.deepEqual(targetNames(boardWith({ h4: "P", g5: "p" }), "h4"), ["g5", "h5"]);
});

test("premove targets: a pawn can reach the last rank", () => {
  assert.deepEqual(targetNames(boardWith({ a7: "P", b8: "n" }), "a7"), ["a8", "b8"]);
});

test("own pieces depend on the side you play", () => {
  assert.equal(isOwnPiece("P", "w"), true);
  assert.equal(isOwnPiece("p", "w"), false);
  assert.equal(isOwnPiece("p", "b"), true);
  assert.equal(isOwnPiece("K", "b"), false);
  assert.equal(isOwnPiece(".", "w"), false);
  assert.equal(isOwnPiece(".", "b"), false);
});

test("results are judged from the side you play", () => {
  assert.equal(gameResult("checkmate", "w", "b").youWon, true);
  assert.equal(gameResult("checkmate", "w", "b").title, "You won");
  assert.equal(gameResult("checkmate", "b", "b").youWon, false);
  assert.equal(gameResult("checkmate", "b", "b").title, "Vajra won");
  assert.equal(gameResult("checkmate", "b", "w").youWon, true);
  assert.equal(gameResult("stalemate", "b", "b").draw, true);
});
