import { render, screen } from "@testing-library/react";
import MiningPage from "@/app/mining/page";
it("links to GPU and compatibility mining guides", () => { render(<MiningPage />); expect(screen.getByRole("link", { name: /nvidia/i })).toHaveAttribute("href", "/mining/nvidia"); expect(screen.getByRole("link", { name: /amd/i })).toHaveAttribute("href", "/mining/amd"); expect(screen.getByRole("link", { name: /kryptex/i })).toHaveAttribute("href", "/mining/kryptex"); });
