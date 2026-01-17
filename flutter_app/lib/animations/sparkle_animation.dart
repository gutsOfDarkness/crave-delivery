import 'package:flutter/material.dart';
import 'dart:math';

/// Sparkle particle for animation
class SparkleParticle {
  late Offset position;
  late double opacity;
  late double size;
  final double baseX;
  final double baseY;

  SparkleParticle({
    required this.baseX,
    required this.baseY,
  }) {
    position = Offset(baseX, baseY);
    opacity = 0;
    size = 4 + Random().nextDouble() * 2;
  }
}

/// Custom sparkle animation widget
class SparkleAnimation extends StatefulWidget {
  final bool isLeft;
  final Duration duration;

  const SparkleAnimation({
    Key? key,
    this.isLeft = true,
    this.duration = const Duration(seconds: 3),
  }) : super(key: key);

  @override
  State<SparkleAnimation> createState() => _SparkleAnimationState();
}

class _SparkleAnimationState extends State<SparkleAnimation>
    with TickerProviderStateMixin {
  late AnimationController _controller;
  final List<SparkleParticle> particles = [];

  @override
  void initState() {
    super.initState();
    _generateParticles();
    _controller = AnimationController(
      duration: widget.duration,
      vsync: this,
    )..repeat();
  }

  void _generateParticles() {
    particles.clear();
    final random = Random();
    for (int i = 0; i < 15; i++) {
      particles.add(
        SparkleParticle(
          baseX: random.nextDouble() * 80,
          baseY: random.nextDouble() * 400,
        ),
      );
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _controller,
      builder: (context, child) {
        return CustomPaint(
          painter: SparklePainter(
            animationValue: _controller.value,
            particles: particles,
            isLeft: widget.isLeft,
          ),
          size: Size.infinite,
        );
      },
    );
  }
}

/// Custom painter for sparkles
class SparklePainter extends CustomPainter {
  final double animationValue;
  final List<SparkleParticle> particles;
  final bool isLeft;

  SparklePainter({
    required this.animationValue,
    required this.particles,
    required this.isLeft,
  });

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..strokeWidth = 1.5
      ..strokeCap = StrokeCap.round;

    for (int i = 0; i < particles.length; i++) {
      final particle = particles[i];
      
      // Create wave motion
      final progress = (animationValue + (i / particles.length)) % 1.0;
      final opacity = sin(progress * pi) * 0.8;
      
      if (opacity > 0) {
        // Vertical floating motion
        final floatOffset = sin(progress * pi * 2) * 20;
        
        final x = isLeft 
            ? particle.baseX 
            : size.width - particle.baseX;
        final y = particle.baseY + floatOffset;

        // Draw sparkle star
        paint.color = Colors.white.withOpacity(opacity);
        _drawStar(canvas, Offset(x, y), particle.size, paint);

        // Optional: Add glow effect
        paint.color = Colors.amber.withOpacity(opacity * 0.5);
        _drawStar(canvas, Offset(x, y), particle.size * 1.5, paint);
      }
    }
  }

  void _drawStar(Canvas canvas, Offset center, double size, Paint paint) {
    final path = Path();
    final points = <Offset>[];
    
    // Create 5-pointed star
    for (int i = 0; i < 5; i++) {
      final angle1 = (i * 4 * pi) / 5 - pi / 2;
      final angle2 = ((i + 1) * 4 * pi) / 5 - pi / 2;
      
      points.add(Offset(
        center.dx + size * cos(angle1),
        center.dy + size * sin(angle1),
      ));
      points.add(Offset(
        center.dx + size * 0.5 * cos(angle1 + angle2) / 2,
        center.dy + size * 0.5 * sin(angle1 + angle2) / 2,
      ));
    }
    
    path.addPolygon(points, true);
    canvas.drawPath(path, paint);
  }

  @override
  bool shouldRepaint(SparklePainter oldDelegate) => true;
}
