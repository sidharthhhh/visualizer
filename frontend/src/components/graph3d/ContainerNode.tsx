import { useRef, useMemo } from 'react';
import { useFrame } from '@react-three/fiber';
import * as THREE from 'three';

interface ContainerNodeProps {
  position: [number, number, number];
  name: string;
  state: string;
  onClick?: () => void;
}

export default function ContainerNode({ position, state, onClick }: ContainerNodeProps) {
  const meshRef = useRef<THREE.Mesh>(null);
  const glowRef = useRef<THREE.Mesh>(null);

  const color = useMemo(() => {
    switch (state) {
      case 'running': return '#00d4ff';
      case 'stopped': return '#444444';
      case 'paused': return '#ffaa00';
      case 'restarting': return '#ffaa00';
      case 'dead': return '#ff4444';
      default: return '#00d4ff';
    }
  }, [state]);

  useFrame((_, delta) => {
    if (meshRef.current) {
      meshRef.current.rotation.y += delta * 0.5;
    }
    if (glowRef.current) {
      glowRef.current.scale.setScalar(1 + Math.sin(Date.now() * 0.003) * 0.1);
    }
  });

  return (
    <group position={position} onClick={onClick}>
      <mesh ref={meshRef}>
        <boxGeometry args={[0.8, 0.5, 0.8]} />
        <meshStandardMaterial
          color={color}
          metalness={0.5}
          roughness={0.3}
          emissive={color}
          emissiveIntensity={0.3}
        />
      </mesh>

      <mesh ref={glowRef}>
        <sphereGeometry args={[0.6, 16, 16]} />
        <meshBasicMaterial
          color={color}
          transparent
          opacity={0.15}
        />
      </mesh>

      <mesh position={[0, -0.35, 0]}>
        <cylinderGeometry args={[0.4, 0.4, 0.05, 32]} />
        <meshStandardMaterial
          color="#111122"
          metalness={0.8}
          roughness={0.2}
        />
      </mesh>
    </group>
  );
}
