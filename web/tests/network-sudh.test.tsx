import { render, screen } from "@testing-library/react";
import SudhPage from "@/app/sudh/page";
it("shows current pre-mainnet parameters", () => { render(<SudhPage />); expect(screen.getByText("51,000,000,000 SUDH")).toBeInTheDocument(); expect(screen.getByText("50 SUDH")).toBeInTheDocument(); expect(screen.getByText("60 seconds")).toBeInTheDocument(); expect(screen.getByText(/subject to change before mainnet/i)).toBeInTheDocument(); });
