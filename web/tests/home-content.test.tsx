import { render, screen } from "@testing-library/react";
import Home from "@/app/page";
it("routes users into the ecosystem", () => { render(<Home />); expect(screen.getByRole("link", { name: /mine sudharma/i })).toHaveAttribute("href", "/mining"); expect(screen.getByRole("link", { name: /build on sudharma/i })).toHaveAttribute("href", "/developers"); expect(screen.getAllByRole("link", { name: /downloads/i }).length).toBeGreaterThan(0); });
