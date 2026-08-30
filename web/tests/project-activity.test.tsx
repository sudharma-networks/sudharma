import { render, screen } from "@testing-library/react";
import { ProjectActivity } from "@/components/project-activity";

it("shows synchronized public GitHub activity without production claims", () => {
  render(<ProjectActivity />);
  expect(screen.getByText(/public github activity/i)).toBeInTheDocument();
  expect(screen.getByText(/wallet-testnet-0\.1\.3/i)).toBeInTheDocument();
  expect(screen.getByText(/does not mean a feature is production-ready/i)).toBeInTheDocument();
});
