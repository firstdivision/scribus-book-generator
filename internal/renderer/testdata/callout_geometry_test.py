import importlib.util
import random
import sys
import unittest


spec = importlib.util.spec_from_file_location("renderer", sys.argv.pop())
renderer = importlib.util.module_from_spec(spec)
spec.loader.exec_module(renderer)


class CalloutGeometryTest(unittest.TestCase):
	def test_bands_around_image(self):
		area = renderer._callout_available_area
		content = (10, 10, 200, 300)
		image = (110, 100, 100, 100)
		self.assertEqual(area(content, 20, image, 0, 5, 0), (10, 20, 200, 80))
		self.assertEqual(area(content, 100, image, 0, 5, 0), (10, 100, 100, 100))
		self.assertEqual(area(content, 200, image, 0, 5, 0), (10, 200, 200, 110))
		self.assertEqual(area(content, 100, (10, 100, 200, 100), 0, 5, 0), (10, 200, 200, 110))

	def test_padding_indent_and_stroke(self):
		area = renderer._callout_available_area
		self.assertEqual(area((0, 0, 100, 100), 0, (50, 20, 50, 40), 5, 2, 2), (6, 1, 88, 18))
		self.assertEqual(area((0, 0, 100, 100), 19, (50, 20, 50, 40), 5, 2, 2), (6, 19, 43, 42))
		self.assertIsNone(area((0, 0, 100, 100), 98, None, 0, 2, 0))
		with self.assertRaisesRegex(RuntimeError, "too narrow"):
			area((0, 0, 20, 100), 0, None, 5, 5, 0)

	def test_random_areas_never_overlap_image(self):
		rng = random.Random(1024)
		for _ in range(1000):
			ex = rng.uniform(-50, 150)
			ey = rng.uniform(-50, 150)
			ew = rng.uniform(1, 150)
			eh = rng.uniform(1, 150)
			y = rng.uniform(0, 100)
			border = rng.uniform(0, 4)
			area = renderer._callout_available_area((0, 0, 100, 100), y, (ex, ey, ew, eh), 0, 1, border)
			if area is None:
				continue
			x, top, width, height = area
			stroke = border / 2
			self.assertGreaterEqual(top, y)
			self.assertGreater(width, 2)
			self.assertGreater(height, 2)
			self.assertTrue(
				x + width + stroke <= ex + 1e-9
				or x - stroke >= ex + ew - 1e-9
				or top + height + stroke <= ey + 1e-9
				or top - stroke >= ey + eh - 1e-9,
				(area, (ex, ey, ew, eh), border),
			)


if __name__ == "__main__":
	unittest.main()
