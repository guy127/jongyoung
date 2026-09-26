import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import BookingError from "./BookingError";

afterEach(cleanup);

// CLAUDE.md ข้อ 9 เทสต์ 30
describe("BookingError", () => {
  it("DUPLICATE_BOOKING: ไม่บอกว่าร้านเต็ม และมีลิงก์ไปรายการเดิม", () => {
    const { container } = render(
      <BookingError error={{ code: "DUPLICATE_BOOKING", message: "", details: { booking_id: "b-123" } }} />,
    );
    expect(screen.getByText("คุณมีการจองที่ทับช่วงนี้อยู่แล้ว")).toBeInTheDocument();
    expect(container.textContent).not.toMatch(/เต็ม|ที่นั่งไม่พอ/);
    expect(screen.getByRole("link", { name: "ดูรายการจองเดิม" })).toHaveAttribute("href", "/bookings/b-123");
  });

  it("NOT_ENOUGH_SEATS: บอกว่าที่นั่งไม่พอและยังไม่ได้จอง", () => {
    render(<BookingError error={{ code: "NOT_ENOUGH_SEATS", message: "", details: { available: 3, at: "2026-10-10T12:30:00+07:00" } }} />);
    expect(screen.getByText(/ที่นั่งไม่พอ/)).toBeInTheDocument();
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });

  it("RESTAURANT_CLOSED → กล่องสีกลางบอกช่วงปิดและเหตุผล ไม่บอกว่าเต็ม", () => {
    render(<BookingError error={{ code: "RESTAURANT_CLOSED", message: "", details: { reason: "ไฟดับ", start_at: "2026-10-10T18:00:00+07:00", end_at: "2026-10-10T20:00:00+07:00" } }} />);
    const box = screen.getByRole("status");
    expect(box).toHaveTextContent("ร้านปิดในช่วงที่เลือก");
    expect(box).toHaveTextContent("ไฟดับ");
    expect(box).not.toHaveTextContent("เต็ม");
  });
});
