package main

import (
	"math/rand"
	"time"

	codenames "github.com/bcspragu/Codenames"
)

var (
	words      = []string{"acne", "acre", "addendum", "advertise", "aircraft", "aisle", "alligator", "alphabetize", "america", "ankle", "apathy", "applause", "applesauce", "application", "archaeologist", "aristocrat", "arm", "armada", "asleep", "astronaut", "athlete", "atlantis", "aunt", "avocado", "backbone", "bag", "baguette", "bald", "balloon", "banana", "banister", "baseball", "baseboards", "basketball", "bat", "battery", "beach", "beanstalk", "bedbug", "beer", "beethoven", "belt", "bib", "bicycle", "big", "bike", "billboard", "bird", "birthday", "bite", "blacksmith", "blanket", "bleach", "blimp", "blossom", "blueprint", "blunt", "blur", "boa", "boat", "bob", "bobsled", "body", "bomb", "bonnet", "book", "booth", "bowtie", "box", "boy", "brainstorm", "brand", "brave", "bride", "bridge", "broccoli", "broken", "broom", "bruise", "brunette", "bubble", "buddy", "buffalo", "bulb", "bunny", "bus", "buy", "cabin", "cafeteria", "cake", "calculator", "campsite", "can", "canada", "candle", "candy", "cape", "capitalism", "car", "cardboard", "cartography", "cat", "cd", "ceiling", "cell", "century", "chair", "chalk", "champion", "charger", "cheerleader", "chef", "chess", "chew", "chicken", "chime", "china", "chocolate", "church", "circus", "clay", "cliff", "cloak", "clockwork", "clown", "clue", "coach", "coal", "coaster", "cog", "cold", "college", "comfort", "computer", "cone", "constrictor", "continuum", "conversation", "cook", "coop", "cord", "corduroy", "cot", "cough", "cow", "cowboy", "crayon", "cream", "crisp", "criticize", "crow", "cruise", "crumb", "crust", "cuff", "curtain", "cuticle", "czar", "dad", "dart", "dawn", "day", "deep", "defect", "dent", "dentist", "desk", "dictionary", "dimple", "dirty", "dismantle", "ditch", "diver", "doctor", "dog", "doghouse", "doll", "dominoes", "door", "dot", "drain", "draw", "dream", "dress", "drink", "drip", "drums", "dryer", "duck", "dump", "dunk", "dust", "ear", "eat", "ebony", "elbow", "electricity", "elephant", "elevator", "elf", "elm", "engine", "england", "ergonomic", "escalator", "eureka", "europe", "evolution", "extension", "eyebrow", "fan", "fancy", "fast", "feast", "fence", "feudalism", "fiddle", "figment", "finger", "fire", "first", "fishing", "fix", "fizz", "flagpole", "flannel", "flashlight", "flock", "flotsam", "flower", "flu", "flush", "flutter", "fog", "foil", "football", "forehead", "forever", "fortnight", "france", "freckle", "freight", "fringe", "frog", "frown", "gallop", "game", "garbage", "garden", "gasoline", "gem", "ginger", "gingerbread", "girl", "glasses", "goblin", "gold", "goodbye", "grandpa", "grape", "grass", "gratitude", "gray", "green", "guitar", "gum", "gumball", "hair", "half", "handle", "handwriting", "hang", "happy", "hat", "hatch", "headache", "heart", "hedge", "helicopter", "hem", "hide", "hill", "hockey", "homework", "honk", "hopscotch", "horse", "hose", "hot", "house", "houseboat", "hug", "humidifier", "hungry", "hurdle", "hurt", "hut", "ice", "implode", "inn", "inquisition", "intern", "internet", "invitation", "ironic", "ivory", "ivy", "jade", "japan", "jeans", "jelly", "jet", "jig", "jog", "journal", "jump", "key", "killer", "kilogram", "king", "kitchen", "kite", "knee", "kneel", "knife", "knight", "koala", "lace", "ladder", "ladybug", "lag", "landfill", "lap", "laugh", "laundry", "law", "lawn", "lawnmower", "leak", "leg", "letter", "level", "lifestyle", "ligament", "light", "lightsaber", "lime", "lion", "lizard", "log", "loiterer", "lollipop", "loveseat", "loyalty", "lunch", "lunchbox", "lyrics", "machine", "macho", "mailbox", "mammoth", "mark", "mars", "mascot", "mast", "matchstick", "mate", "mattress", "mess", "mexico", "midsummer", "mine", "mistake", "modern", "mold", "mom", "monday", "money", "monitor", "monster", "mooch", "moon", "mop", "moth", "motorcycle", "mountain", "mouse", "mower", "mud", "music", "mute", "nature", "negotiate", "neighbor", "nest", "neutron", "niece", "night", "nightmare", "nose", "oar", "observatory", "office", "oil", "old", "olympian", "opaque", "opener", "orbit", "organ", "organize", "outer", "outside", "ovation", "overture", "pail", "paint", "pajamas", "palace", "pants", "paper", "paper", "park", "parody", "party", "password", "pastry", "pawn", "pear", "pen", "pencil", "pendulum", "penis", "penny", "pepper", "personal", "philosopher", "phone", "photograph", "piano", "picnic", "pigpen", "pillow", "pilot", "pinch", "ping", "pinwheel", "pirate", "plaid", "plan", "plank", "plate", "platypus", "playground", "plow", "plumber", "pocket", "poem", "point", "pole", "pomp", "pong", "pool", "popsicle", "population", "portfolio", "positive", "post", "princess", "procrastinate", "protestant", "psychologist", "publisher", "punk", "puppet", "puppy", "push", "puzzle", "quarantine", "queen", "quicksand", "quiet", "race", "radio", "raft", "rag", "rainbow", "rainwater", "random", "ray", "recycle", "red", "regret", "reimbursement", "retaliate", "rib", "riddle", "rim", "rink", "roller", "room", "rose", "round", "roundabout", "rung", "runt", "rut", "sad", "safe", "salmon", "salt", "sandbox", "sandcastle", "sandwich", "sash", "satellite", "scar", "scared", "school", "scoundrel", "scramble", "scuff", "seashell", "season", "sentence", "sequins", "set", "shaft", "shallow", "shampoo", "shark", "sheep", "sheets", "sheriff", "shipwreck", "shirt", "shoelace", "short", "shower", "shrink", "sick", "siesta", "silhouette", "singer", "sip", "skate", "skating", "ski", "slam", "sleep", "sling", "slow", "slump", "smith", "sneeze", "snow", "snuggle", "song", "space", "spare", "speakers", "spider", "spit", "sponge", "spool", "spoon", "spring", "sprinkler", "spy", "square", "squint", "stairs", "standing", "star", "state", "stick", "stockholder", "stoplight", "stout", "stove", "stowaway", "straw", "stream", "streamline", "stripe", "student", "sun", "sunburn", "sushi", "swamp", "swarm", "sweater", "swimming", "swing", "tachometer", "talk", "taxi", "teacher", "teapot", "teenager", "telephone", "ten", "tennis", "thief", "think", "throne", "through", "thunder", "tide", "tiger", "time", "tinting", "tiptoe", "tiptop", "tired", "tissue", "toast", "toilet", "tool", "toothbrush", "tornado", "tournament", "tractor", "train", "trash", "treasure", "tree", "triangle", "trip", "truck", "tub", "tuba", "tutor", "television", "twang", "twig", "type", "unemployed", "upgrade", "vest", "vision", "wag", "water", "watermelon", "wax", "wedding", "weed", "welder", "whatever", "wheelchair", "whiplash", "whisk", "whistle", "white", "wig", "will", "windmill", "winter", "wish", "wolf", "wool", "world", "worm", "wristwatch", "yardstick", "zamboni", "zen", "zero", "zipper", "zone", "zoo"}
	baseAgents = []codenames.Agent{
		codenames.RedAgent,
		codenames.RedAgent,
		codenames.RedAgent,
		codenames.RedAgent,
		codenames.RedAgent,
		codenames.RedAgent,
		codenames.RedAgent,
		codenames.RedAgent,
		codenames.BlueAgent,
		codenames.BlueAgent,
		codenames.BlueAgent,
		codenames.BlueAgent,
		codenames.BlueAgent,
		codenames.BlueAgent,
		codenames.BlueAgent,
		codenames.BlueAgent,
		codenames.Bystander,
		codenames.Bystander,
		codenames.Bystander,
		codenames.Bystander,
		codenames.Bystander,
		codenames.Bystander,
		codenames.Bystander,
		codenames.Assassin,
	}
)

func board(starter codenames.Team) *codenames.Board {
	rand.Seed(time.Now().UnixNano())

	used := make(map[string]struct{})
	agents := make([]codenames.Agent, len(baseAgents))
	copy(agents, baseAgents)

	switch starter {
	case codenames.RedTeam:
		agents = append(agents, codenames.RedAgent)
	case codenames.BlueTeam:
		agents = append(agents, codenames.BlueAgent)
	}

	// Pick words at random from our list.
	for len(used) < codenames.Size {
		used[words[rand.Intn(len(words))]] = struct{}{}
	}

	var selected []string
	for word := range used {
		selected = append(selected, word)
	}

	var cards []codenames.Card
	for i, idx := range rand.Perm(len(agents)) {
		cards = append(cards, codenames.Card{
			Agent:    agents[idx],
			Codename: selected[i],
		})
	}

	return &codenames.Board{Cards: cards}
}
