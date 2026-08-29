import { render, screen } from "@testing-library/react";
import ExplorerPage from "@/app/explorer/page";
it("does not fabricate explorer metrics", () => { render(<ExplorerPage />); expect(screen.getAllByText(/in development|planned|testnet/i).length).toBeGreaterThan(0); expect(screen.queryByText(/live blocks: 12345/i)).not.toBeInTheDocument(); });
