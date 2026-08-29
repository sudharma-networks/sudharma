import { render, screen } from "@testing-library/react";
import Home from "@/app/page";

test("shows a public website visitor counter on the homepage", () => {
  render(<Home />);
  expect(screen.getByText("Website Visitors")).toBeInTheDocument();
});
