import { render, screen } from "@testing-library/react";
import Home from "@/app/page";

test("renders the Sudharma homepage identity", () => {
  render(<Home />);
  expect(screen.getByRole("heading", { name: /open blockchain/i })).toBeInTheDocument();
});
